package api

import (
	"context"
	"fmt"

	"github.com/denkoren/mi-labs-test/internal/core"
	"github.com/denkoren/mi-labs-test/internal/interconnect/docker"
	"github.com/denkoren/mi-labs-test/internal/interconnect/registry"
	apipb "github.com/denkoren/mi-labs-test/proto/api/v1"
)

type Server struct {
	apipb.UnimplementedZapuskatorAPIServer

	registry *registry.ContainerRegistry
	docker *docker.ContainerManager
}

func (s *Server) Calculate(ctx context.Context, request *apipb.Calculate_Request) (*apipb.Calculate_Response, error) {
	c, err := s.registry.ExistingOrNewByParams(core.ContainerParams{
		Seed:  request.GetParams().Seed,
		Input: request.GetParams().Input,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to register new container: %v", err)
	}
	defer c.Release()

	if c.Status == core.ContainerStatusNew {
		cc, err := s.docker.StartContainer(c.ContainerInfo.Params)
		if err != nil {
			return nil, fmt.Errorf("failed to start new container: %w", err)
		}

		c.ContainerInfo = cc
		c.Save()
	}

	s.waitForContainer(c, timeout)

	return &apipb.Calculate_Response{Data: []byte("Here is my response")}, nil
}

func (s *Server) waitForContainer(c *registry.ContainerInfo) error {
	return fmt.Errorf("not implemented")
}
