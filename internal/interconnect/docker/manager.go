package docker

import (
	"context"
	"log"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	dclient "github.com/docker/docker/client"

	"github.com/denkoren/mi-labs-test/internal/core"
)

type ManagerConfig struct {
	Host           string
	RequestTimeout time.Duration
	ImageTag       string
}

type Manager struct {
	config ManagerConfig
	docker *dclient.Client
}

func NewManager(config ManagerConfig) (*Manager, error) {
	docker, err := dclient.NewClientWithOpts(
		//dclient.WithHost(config.Host),
		dclient.WithAPIVersionNegotiation(),
		dclient.WithTimeout(config.RequestTimeout),
	)

	if err != nil {
		return nil, err
	}

	return &Manager{
		config: config,
		docker: docker,
	}, nil
}

func (m *Manager) StartContainer(ctx context.Context, params core.ContainerParams) (core.ContainerInfo, error) {
	log.Printf("Starting container for seed: %s", params.Seed)

	createResult, err := m.docker.ContainerCreate(ctx, &container.Config{
		Image: m.config.ImageTag,
		Tty:   false,
	}, nil, nil, nil, "")

	containerID := createResult.ID

	if err != nil {
		return core.ContainerInfo{}, err
	}

	err = m.docker.ContainerStart(ctx, containerID, types.ContainerStartOptions{})
	if err != nil {
		return core.ContainerInfo{}, err
	}

	dContainerInfo, err := m.docker.ContainerInspect(ctx, containerID)
	if err != nil {
		return core.ContainerInfo{}, err
	}

	c := core.NewContainerInfo(
		containerID,
		dContainerInfo.NetworkSettings.IPAddress,
		params,
	)
	c.Status = core.ContainerStatusStarting
	c.Scheduled = time.Now()

	return c, nil
}

func (m *Manager) IsContainerRunning(ctx context.Context, id string) (bool, error) {
	info, err := m.docker.ContainerInspect(ctx, id)
	if err != nil {
		return false, err
	}

	return info.State.Running, nil
}

func (m *Manager) IsContainerStopped(ctx context.Context, id string) (bool, error) {
	info, err := m.docker.ContainerInspect(ctx, id)
	if err != nil {
		return false, err
	}

	// FIXME: not sure about correct way to detect stopped container
	return info.State.Status == "exited", nil
}

func (m *Manager) StopContainer(ctx context.Context, id string) error {
	log.Printf("Stopping container: id: %s", id)
	return m.docker.ContainerStop(ctx, id, &m.config.RequestTimeout)
}
