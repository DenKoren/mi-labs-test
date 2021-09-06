package util

import (
	"context"
	"reflect"
	"sync"
)

type Multicontext struct {
	outContext       context.Context
	cancelOutContext context.CancelFunc
	isCanceled       bool

	inContexts []context.Context
	inCancels  []context.CancelFunc

	cancelStubContext context.CancelFunc
	lock              sync.Mutex
}

func NewMulticontext() *Multicontext {
	outCtx, cancelOutCtx := context.WithCancel(context.Background())

	stubCtx, cancelStubCtx := context.WithCancel(outCtx)
	inContexts := make([]context.Context, 1)
	inCancels := make([]context.CancelFunc, 1)
	inContexts[0] = stubCtx
	inCancels[0] = cancelStubCtx

	mCtx := &Multicontext{
		outContext:        outCtx,
		cancelOutContext:  cancelOutCtx,
		cancelStubContext: cancelStubCtx,

		inContexts: inContexts,
		inCancels:  inCancels,
	}

	go mCtx.waitAllCanceled()
	return mCtx
}

func (m *Multicontext) Ctx() context.Context {
	return m.outContext
}

func (m *Multicontext) Cancel() {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.cancelOutContext()
	m.isCanceled = true
	for _, cancel := range m.inCancels {
		cancel()
	}
}

func (m *Multicontext) AddCtx(ctxs ...context.Context) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.isCanceled {
		return context.Canceled
	}

	for _, c := range ctxs {
		ctx, cancel := context.WithCancel(c)
		m.inContexts = append(m.inContexts, ctx)
		m.inCancels = append(m.inCancels, cancel)
	}

	m.cancelStubContext()
	return nil
}

func (m *Multicontext) removeInCtx(i int) {
	last := len(m.inContexts) - 1
	if i < 0 || last < 0 {
		// We try to remove non-existent element or the slice is empty
		return
	}

	// Replace the deleted ctx with the last from slice
	m.inContexts[i] = m.inContexts[last]
	m.inCancels[i] = m.inCancels[last]

	// Reduce slice size
	m.inContexts = m.inContexts[:last]
	m.inCancels = m.inCancels[:last]
}

func (m *Multicontext) waitAllCanceled() {
	selectCases := func() []reflect.SelectCase {
		cases := make([]reflect.SelectCase, len(m.inContexts))
		for i, ctx := range m.inContexts {
			cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ctx.Done())}
		}
		return cases
	}

	m.lock.Lock()
	cases := selectCases()
	m.lock.Unlock()

	for {
		chosen, _, _ := reflect.Select(cases)

		m.lock.Lock()
		m.removeInCtx(chosen)

		if len(m.inContexts) != 0 {
			cases = selectCases()
			m.lock.Unlock()
			// Some contexts are not done yet, go to the next iteration
			continue
		}

		// All contexts are done.
		m.cancelOutContext()
		m.isCanceled = true
		m.lock.Unlock()
		return
	}
}
