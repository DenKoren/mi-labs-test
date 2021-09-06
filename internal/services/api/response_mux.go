package api

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/denkoren/mi-labs-test/internal/util"
	"github.com/docker/go-units"
)

const defaultDedupBufferSize = 100 * units.KiB

type responseDuplicator struct {
	request    *http.Request
	requestCtx *util.Multicontext

	requesters []io.Writer
	closers    []io.Closer
	errors     []chan<- error

	started  bool
	finished bool
	lock     sync.Mutex
}

func newMultiRequest(method string, url string, body io.Reader) (*responseDuplicator, error) {
	reqCtx := util.NewMulticontext()

	r, err := http.NewRequestWithContext(
		reqCtx.Ctx(),
		method,
		url,
		body,
	)
	if err != nil {
		return nil, err
	}

	return &responseDuplicator{
		request:    r,
		requestCtx: reqCtx,
		requesters: make([]io.Writer, 0, 1),
		closers:    make([]io.Closer, 0, 1),
		errors:     make([]chan<- error, 0, 1),
	}, nil
}

func (r *responseDuplicator) registerReader(ctx context.Context) (io.ReadCloser, <-chan error, error) {
	log.Printf("[RMUX] registering new reader for request '%s'", r.request.URL)

	err := r.requestCtx.AddCtx(ctx)
	if err != nil {
		log.Printf("[RMUX] new reader's context registration failed for '%s': %s", r.request.URL, err.Error())
		return nil, nil, err
	}

	pr, pw := io.Pipe()
	errCh := make(chan error, 1)
	r.requesters = append(r.requesters, pw)
	r.closers = append(r.closers, pw)
	r.errors = append(r.errors, errCh)

	return pr, errCh, nil
}

// Thread-safe
func (r *responseDuplicator) do() {
	r.lock.Lock()
	if r.started {
		r.lock.Unlock()
		return
	}
	r.started = true
	r.lock.Unlock()

	var (
		response *http.Response
		err      error = nil
	)

	log.Printf("[RMUX] request to '%s' started", r.request.URL)

	response, err = http.DefaultClient.Do(r.request)
	if err != nil {
		r.sendError(err)
		return
	}
	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("container responded with code '%d: %s'", response.StatusCode, response.Status)
		r.sendError(err)
		return
	}

	r.lock.Lock()
	defer r.lock.Unlock()

	log.Printf("[RMUX] request to '%s' finished, writing response data...", r.request.URL)

	r.finished = true
	r.sendError(nil) // Send 'nil' error to clients so they can read response

	mul := io.MultiWriter(r.requesters...)
	_, err = io.CopyBuffer(mul, response.Body, make([]byte, defaultDedupBufferSize))

	log.Printf("[RMUX] response from '%s' written. Err: %v", r.request.URL, err)

	log.Printf("[RMUX] closing '%d' response writers for '%s'...", len(r.closers), r.request.URL)
	for _, cl := range r.closers {
		err = cl.Close()
		if err != nil {
			log.Printf("[RMUX] failed to close '%s' response writer: %s", r.request.URL, err.Error())
		}
	}

	log.Printf("[RMUX] response writers for '%s' closed: %d", r.request.URL, len(r.closers))
	return
}

func (r *responseDuplicator) sendError(err error) {
	for _, ch := range r.errors {
		ch <- err
		close(ch)
	}
}

type requestIndex map[string]*responseDuplicator

type responseMux struct {
	activeRequests requestIndex

	indexLock sync.Mutex
}

func newResponseMux() *responseMux {
	return &responseMux{
		activeRequests: make(requestIndex),
	}
}

// Thread-safe
func (m *responseMux) getRequest(ctx context.Context, method, url string) (io.Reader, <-chan error, error) {
	m.indexLock.Lock()
	defer m.indexLock.Unlock()

	var (
		req    *responseDuplicator
		reader io.Reader
		ok     bool
		err    error
		errCh  <-chan error
	)

	requestIndexID := m.getRequestIndexID(method, url)
	req, ok = m.activeRequests[requestIndexID]
	if !ok {
		// No active request exist. Create new.
		return m.runNewMultiRequest(ctx, method, url)
	}

	req.lock.Lock()
	defer req.lock.Unlock()

	if req.finished {
		// Request already done. We need new one.
		return m.runNewMultiRequest(ctx, method, url)
	}

	reader, errCh, err = req.registerReader(ctx)
	if err == context.Canceled {
		// Request was canceled. We need new one.
		return m.runNewMultiRequest(ctx, method, url)
	}

	log.Printf("[RMUX] got new reader for existing request to '%s'", url)
	return reader, errCh, err
}

func (m *responseMux) runNewMultiRequest(ctx context.Context, method, url string) (io.Reader, <-chan error, error) {
	log.Printf("[RMUX] making new miltirequest for '%s'", url)

	req, err := newMultiRequest(method, url, nil)
	if err != nil {
		return nil, nil, err
	}

	requestIndexID := m.getRequestIndexID(method, url)
	m.activeRequests[requestIndexID] = req

	reader, errCh, err := req.registerReader(ctx)
	go req.do()

	log.Printf("[RMUX] new miltirequest for '%s' created", url)
	return reader, errCh, err
}

func (m *responseMux) getRequestIndexID(method, url string) string {
	return fmt.Sprintf("%s:%s", method, url)
}
