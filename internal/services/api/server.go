package api

import (
	"context"
	"errors"
	"fmt"
	"log"
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
	docker   *docker.ContainerManager
}

func NewServer(config Config, reg *registry.ContainerRegistry, dock *docker.ContainerManager) (*Server, error) {
	return &Server{
		config: config,

		registry: reg,
		docker:   dock,
	}, nil
}

func (s *Server) Calculate(ctx context.Context, request *apipb.Calculate_Request) (*apipb.Calculate_Response, error) {
	c, err := s.registry.ExistingOrNewByParams(core.ContainerParams{
		Seed:  request.GetParams().Seed,
		Input: request.GetParams().Input,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to register new container: %v", err)
	}

	err = s.startContainer(ctx, c)
	if err != nil {
		return nil, err
	}

	err = s.waitForContainer(ctx, c)
	if err != nil {
		return nil, err
	}

	// FIXME: is the data huge? Prefer stream here.
	return &apipb.Calculate_Response{Data: []byte("Here is my response")}, nil
}

func (s *Server) startContainer(ctx context.Context, c *registry.ContainerInfo) error {
	if c.Status != core.ContainerStatusNew {
		// Container does not need starting
		return nil
	}

	err := c.ToStarting(
		func(_ core.ContainerStatus) error {
			return s.startContainerHook(ctx, c)
		},
	)

	if err == nil {
		// We successfully scheduled the container
		return nil
	}

	if errors.Is(err, registry.ErrTransitionNotAllowed) {
		if c.Status == core.ContainerStatusStarting {
			// Someone already took container and scheduled it.
			return nil
		}
	}

	return err
}

func (s *Server) startContainerHook(ctx context.Context, c *registry.ContainerInfo) error {
	log.Printf("Container status is '%s', starting new container", c.Status.String())
	cc, err := s.docker.StartContainer(ctx, c.ContainerInfo.Params)
	if err != nil {
		return fmt.Errorf("failed to start new container: %w", err)
	}

	c.ID = cc.ID
	c.Addr = cc.Addr

	log.Printf("Container '%s' was scheduled", c.ID)
	return nil
}

func (s *Server) waitForContainer(ctx context.Context, c *registry.ContainerInfo) error {
	if c.Status == core.ContainerStatusReady {
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
			err := s.refreshContainerStatus(waitCtx, c)
			if err != nil {
				return fmt.Errorf("failed to wait for container '%s' start: %w", c.ID, err)
			}
			if !c.Status.IsAvailable() {
				return fmt.Errorf("something went wrong with container '%s' and it moved to '%s' status", c.ID, c.Status.String())
			}
			if c.Status == core.ContainerStatusReady {
				return nil
			}

		case <-waitCtx.Done():
			return fmt.Errorf("failed to wait for container '%s' start: %w", c.ID, waitCtx.Err())
		}
	}
}

func (s *Server) refreshContainerStatus(ctx context.Context, c *registry.ContainerInfo) error {
	return fmt.Errorf("'refreshContainerStatus' not implemented")
}
