package api

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/denkoren/mi-labs-test/internal/core"
	"github.com/denkoren/mi-labs-test/internal/interconnect/docker"
	"github.com/denkoren/mi-labs-test/internal/interconnect/registry"
	apipb "github.com/denkoren/mi-labs-test/proto/api/v1"
)

type Config struct {
	ContainerWaitTimeout time.Duration
}

type Server struct {
	apipb.UnimplementedZapuskatorAPIServer

	config Config

	registry  *registry.ContainerRegistry
	docker    *docker.Manager
	requester *responseMux
}

func NewServer(config Config, reg *registry.ContainerRegistry, dock *docker.Manager) (*Server, error) {
	return &Server{
		config: config,

		registry:  reg,
		docker:    dock,
		requester: newResponseMux(),
	}, nil
}

func (s *Server) Calculate(ctx context.Context, request *apipb.Calculate_Request) (*apipb.Calculate_Response, error) {
	container, err := s.registry.ExistingOrNewByParams(core.ContainerParams{
		Seed: request.GetParams().Seed,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to register new container: %v", err)
	}

	go s.refreshContainerLastUsed(ctx, container)

	err = s.createContainer(ctx, container)
	if err != nil {
		return nil, err
	}

	err = s.startContainer(ctx, container)
	if err != nil {
		return nil, err
	}

	err = s.waitForContainer(ctx, container)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("http://%s:8080/calculate/%s", container.Addr, request.Params.Input)
	log.Printf("[API] starting request to '%s'", url)
	reader, errCh, err := s.requester.getRequest(
		ctx,
		http.MethodGet,
		url,
	)

	if err != nil {
		return nil, err
	}

	log.Printf("[API] got reader and err chan for '%s'", url)
	err = <-errCh
	if err != nil {
		return nil, err
	}

	log.Printf("[API] got response from '%s', reading data...", url)
	data, err := ioutil.ReadAll(reader)

	// FIXME: is the data huge? Prefer stream here.
	return &apipb.Calculate_Response{Data: data}, err
}

func (s *Server) createContainer(ctx context.Context, container *registry.ContainerInfo) error {
	log.Printf("[API] creating container for seed '%s'", container.Params.Seed)

	err := container.ToCreated(
		func(c *registry.ContainerInfo, _ core.ContainerStatus) error {
			if c.Status != core.ContainerStatusNew {
				// Container was already created, nothing to do
				return nil
			}

			id, err := s.docker.CreateContainer(ctx, container.Params)
			if err != nil {
				return err
			}

			container.ID = id
			return nil
		},
		logTransition,
	)

	if errors.Is(err, registry.ErrTransitionNotAllowed) {
		if container.Status.WasCreated() {
			// Another thread already created container
			return nil
		}
	}

	_ = container.ToFailed(logTransition)
	return err
}

func (s *Server) startContainer(ctx context.Context, container *registry.ContainerInfo) error {
	log.Printf("[API] starting container '%s'", container.ID)

	err := container.ToStarting(
		func(c *registry.ContainerInfo, _ core.ContainerStatus) error {
			if container.Status.IsActive() {
				// container already was started
				return nil
			}

			if !container.Status.IsStartable() {
				// We can't start this container
				return fmt.Errorf("can't start container in status %s", container.Status)
			}

			addr, err := s.docker.StartContainer(ctx, container.ID)
			if err != nil {
				return err
			}
			container.Addr = addr
			return nil
		},
		logTransition,
	)

	if errors.Is(err, registry.ErrTransitionNotAllowed) {
		if container.Status.IsActive() {
			// Another thread already started container
			return nil
		}
	}

	return err
}

func (s *Server) waitForContainer(ctx context.Context, container *registry.ContainerInfo) error {
	subscription := container.Subscribe()
	defer subscription.Unsubscribe()

	log.Printf("[API] waiting for container '%s' start", container.ID)

	if container.Status == core.ContainerStatusReady {
		// Container ready for work
		// We need to check this before reading the channel to make sure we did not miss the
		// event while subscribing
		return nil
	}

	waitCtx, cancel := context.WithTimeout(ctx, s.config.ContainerWaitTimeout)
	defer cancel()

	for {
		select {
		case <-subscription.C:
			if container.Status == core.ContainerStatusReady {
				// Container ready for work
				return nil
			}
		case <-waitCtx.Done():
			return fmt.Errorf("failed to wait for container '%s' start: %w", container.ID, waitCtx.Err())
		}
	}
}

func (s *Server) ignoreErr(err error, ignored ...error) error {
	for _, toIgnore := range ignored {
		if errors.Is(err, toIgnore) {
			return nil
		}
	}

	return err
}

func (s *Server) refreshContainerLastUsed(ctx context.Context, container *registry.ContainerInfo) {
	// FIXME: use 1/2 of container idle timeout here
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			container.UpdateLastUsed()
		}
	}
}

func logTransition(c *registry.ContainerInfo, newStatus core.ContainerStatus) error {
	log.Printf("[API] container '%s' transitioned from '%s' to '%s'", c.ID, c.Status.String(), newStatus.String())
	return nil
}
