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

	registry *registry.ContainerRegistry
	docker   *docker.Manager
}

func NewServer(config Config, reg *registry.ContainerRegistry, dock *docker.Manager) (*Server, error) {
	return &Server{
		config: config,

		registry: reg,
		docker:   dock,
	}, nil
}

func (s *Server) Calculate(ctx context.Context, request *apipb.Calculate_Request) (*apipb.Calculate_Response, error) {
	container, err := s.registry.ExistingOrNewByParams(core.ContainerParams{
		Seed:  request.GetParams().Seed,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to register new container: %v", err)
	}

	go s.refreshContainerLastUsed(ctx, container)

	err = s.startContainer(ctx, container)
	if err != nil {
		return nil, err
	}

	err = s.waitForContainer(ctx, container)
	if err != nil {
		return nil, err
	}

	proxyRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("http://%s:8080/calculate/%s", container.Addr, request.Params.Input),
		nil,
	)
	if err != nil {
		return nil, err
	}

	response, err := http.DefaultClient.Do(proxyRequest)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("container responded with code '%d: %s'", response.StatusCode, response.Status)
	}

	data, err := ioutil.ReadAll(response.Body)

	// FIXME: is the data huge? Prefer stream here.
	return &apipb.Calculate_Response{Data: data}, err
}

func (s *Server) startContainer(ctx context.Context, container *registry.ContainerInfo) error {
	if container.Status != core.ContainerStatusNew {
		// Container does not need starting
		return nil
	}

	err := container.ToStarting(
		func(c *registry.ContainerInfo, _ core.ContainerStatus) error {
			return s.startContainerHook(ctx, c)
		},
	)

	if err == nil {
		// We successfully scheduled the container
		return nil
	}

	if errors.Is(err, registry.ErrTransitionNotAllowed) {
		if container.Status == core.ContainerStatusStarting {
			// Someone already took container and scheduled it.
			return nil
		}
	}

	return err
}

func (s *Server) startContainerHook(ctx context.Context, container *registry.ContainerInfo) error {
	log.Printf("Container status is '%s', starting new docker container", container.Status.String())
	cc, err := s.docker.StartContainer(ctx, container.ContainerInfo.Params)
	if err != nil {
		return fmt.Errorf("failed to start new container: %w", err)
	}

	container.ID = cc.ID
	container.Addr = cc.Addr

	err = s.registry.Register(container)
	if err != nil {
		return fmt.Errorf("failed to register new container '%s': %w", container.ID, err)
	}

	log.Printf("Container '%s' was scheduled", container.ID)
	return nil
}

func (s *Server) waitForContainer(ctx context.Context, container *registry.ContainerInfo) error {
	if container.Status == core.ContainerStatusReady {
		// Container ready for work
		return nil
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	waitCtx, cancel := context.WithTimeout(ctx, s.config.ContainerWaitTimeout)
	defer cancel()

	for {
		select {
		case <-ticker.C:
			err := s.refreshContainerStatus(waitCtx, container)
			if container.Status == core.ContainerStatusReady {
				log.Printf("container '%s' is ready", container.ID)
				return nil
			}

			if err != nil {
				return fmt.Errorf("failed to wait for container '%s' start: %w", container.ID, err)
			}
			if !container.Status.IsAvailable() {
				return fmt.Errorf("something went wrong with container '%s' and it moved to '%s' status", container.ID, container.Status.String())
			}

		case <-waitCtx.Done():
			return fmt.Errorf("failed to wait for container '%s' start: %w", container.ID, waitCtx.Err())
		}
	}
}

func (s *Server) refreshContainerStatus(ctx context.Context, container *registry.ContainerInfo) error {
	isRunning, err := s.docker.IsContainerRunning(ctx, container.ID)
	if err != nil {
		log.Printf("can't check docker container '%s' status: %v", container.ID, err)
		return err
	}

	if !isRunning {
		log.Printf("container '%s' is not running yet", container.ID)
		return nil
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("http://%s:8080/health", container.Addr),
		nil,
	)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("container '%s' is not ready yet", container.ID)
		return nil
	}

	return container.ToReady()
}

func (s *Server) refreshContainerLastUsed(ctx context.Context, container *registry.ContainerInfo) {
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
