package docker

import (
	"context"
	"log"
	"time"

	"github.com/denkoren/mi-labs-test/internal/core"
	"github.com/denkoren/mi-labs-test/internal/util"
)

type ContainerManager struct {
}

func NewContainerManager() (*ContainerManager, error) {
	return &ContainerManager{}, nil
}

func (*ContainerManager) StartContainer(ctx context.Context, params core.ContainerParams) (core.ContainerInfo, error) {
	log.Printf("Starting container: seed: %s, input: %s", params.Seed, params.Input)
	c := core.NewContainerInfo(
		util.RandString(10),
		"some.example.addr:8888",
		params,
	)
	c.Status = core.ContainerStatusStarting
	c.Scheduled = time.Now()

	return c, nil
}

func (*ContainerManager) GetContainerInfo(id string) (core.ContainerInfo, error) {
	return core.ContainerInfo{
		ID: id,
		// ...
	}, nil
}

func (*ContainerManager) StopContainer(id string) error {
	log.Printf("Stopping container: id: %s", id)
	return nil
}
