package docker

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/denkoren/mi-labs-test/internal/core"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	dclient "github.com/docker/docker/client"
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

type ContainerState string

const (
	ContainerStateUnknown    ContainerState = "unknown"
	ContainerStateCreated    ContainerState = "created"
	ContainerStateRunning    ContainerState = "running"
	ContainerStatePaused     ContainerState = "paused"
	ContainerStateRestarting ContainerState = "restarting"
	ContainerStateRemoving   ContainerState = "removing"
	ContainerStateExited     ContainerState = "exited"
	ContainerStateDead       ContainerState = "dead"
)

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

func (m *Manager) CreateContainer(ctx context.Context, params core.ContainerParams) (string, error) {
	log.Printf("[Docker] creating container for seed: %s", params.Seed)

	seedEnv := fmt.Sprintf("SEED=%s", params.Seed)

	createResult, err := m.docker.ContainerCreate(ctx, &container.Config{
		Image: m.config.ImageTag,
		Tty:   false,
		Env: []string{seedEnv},
	}, nil, nil, nil, "")

	if err != nil {
		return "", err
	}

	log.Printf("[Docker] container with ID '%s' created for seed %s", createResult.ID, params.Seed)
	return createResult.ID, nil
}

func (m *Manager) StartContainer(ctx context.Context, id string) (string, error) {
	log.Printf("[Docker] starting container '%s'", id)

	err := m.docker.ContainerStart(ctx, id, types.ContainerStartOptions{})
	if err != nil {
		return "", err
	}

	dInfo, err := m.docker.ContainerInspect(ctx, id)
	if err != nil {
		return "", err
	}

	return dInfo.NetworkSettings.IPAddress, nil
}

func (m *Manager) ContainerState(ctx context.Context, id string) (ContainerState, error) {
	info, err := m.docker.ContainerInspect(ctx, id)
	if err != nil {
		return ContainerStateUnknown, err
	}

	return ContainerState(info.State.Status), nil
}

func (m *Manager) StopContainer(ctx context.Context, id string) error {
	log.Printf("[Docker] stopping container '%s'", id)
	return m.docker.ContainerStop(ctx, id, &m.config.RequestTimeout)
}
