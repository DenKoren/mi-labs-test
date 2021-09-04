package background

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/denkoren/mi-labs-test/internal/core"
	"github.com/denkoren/mi-labs-test/internal/interconnect/docker"
	"github.com/denkoren/mi-labs-test/internal/interconnect/registry"
)

type Config struct {
	InactiveContainerTimeout time.Duration
	ContainersCheckInterval time.Duration
}

type Background struct {
	config Config
	registry *registry.ContainerRegistry
	docker *docker.Manager
}

func NewBackground(config Config, registry *registry.ContainerRegistry, docker *docker.Manager) (*Background, error) {
	return &Background{
		config: config,
		registry: registry,
		docker: docker,
	}, nil
}

func (s *Background) Run(ctx context.Context) error {
	wg := &sync.WaitGroup{}

	wg.Add(1)
	go s.watchActiveContainers(ctx, wg)

	wg.Add(1)
	go s.stopInactiveContainers(ctx, wg)


	wg.Wait()
	return nil
}

func (s *Background) watchActiveContainers(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	ticker := time.NewTicker(s.config.ContainersCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			containers, err := s.registry.ActiveContainers()
			if err != nil {
				log.Printf("[BG] failed to load active containers list: %s", err.Error())
				continue
			}

			log.Printf("[BG] detected '%d' active containers", len(containers))
			for _, container := range containers {
				s.updateDockerContainerStatus(ctx, container)
			}

		case <-ctx.Done():
			log.Printf("[BG] task 'watchActiveContainers' context done: %v", ctx.Err())
			return
		}
	}
}

func (s *Background) updateDockerContainerStatus(ctx context.Context, container *registry.ContainerInfo) {
	dState, err := s.docker.ContainerState(ctx, container.ID)
	if err != nil {
		log.Printf("[BG] can't check docker container '%s' status: %v", container.ID, err)
		return
	}

	logErr := func(err error) {
		if err == nil {
			return
		}
		log.Printf("[BG] failed to update '%s' container status: %s", container.ID, err.Error())
	}

	switch dState {
	case docker.ContainerStateRunning:
		logErr(s.containerHealthcheck(ctx, container))
	case docker.ContainerStatePaused:
		logErr(container.ToPaused(logTransition))
	case docker.ContainerStateRestarting:
		logErr(container.ToStarting(logTransition))
	case docker.ContainerStateRemoving,
		docker.ContainerStateExited,
		docker.ContainerStateDead:
		logErr(container.ToStopped(logTransition))
	}
}

func (s *Background) containerHealthcheck(ctx context.Context, container *registry.ContainerInfo) error {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("http://%s:8080/health", container.Addr),
		nil,
	)
	if err != nil {
		log.Printf("[BG] container '%s' healthcheck failed: %s", container.ID, err.Error())
		return container.ToUnreachable(logTransition)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[BG] container '%s' healthcheck failed: %s", container.ID, err.Error())
		return container.ToUnreachable(logTransition)
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("[BG] container '%s' is not ready (healthcheck '%d')", container.ID, resp.StatusCode)
		return container.ToRunning(logTransition)
	}

	log.Printf("[BG] container '%s' is ready (healthcheck '%d')", container.ID, resp.StatusCode)
	return container.ToReady(logTransition)
}

func (s *Background) stopInactiveContainers(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	ticker := time.NewTicker(s.config.ContainersCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			lastUsedBefore := time.Now().Add(-s.config.InactiveContainerTimeout)

			containers, err := s.registry.OldContainers(lastUsedBefore)
			if err != nil {
				log.Printf("[BG] failed to load old containers list")
				continue
			}

			log.Printf("[BG] detected '%d' old containers", len(containers))
			for _, container := range containers {
				err = s.scheduleContainerStop(ctx, container, lastUsedBefore)
				if err != nil {
					log.Printf("[BG] failed to stop container '%s': %v", container.ID, err)
					continue
				}

				log.Printf("[BG] scheduled container '%s' stop", container.ID)
			}

		case <-ctx.Done():
			log.Printf("[BG] task 'stopInactiveContainers' context done: %v", ctx.Err())
			return
		}
	}
}

func (s *Background) scheduleContainerStop(ctx context.Context, container *registry.ContainerInfo, lastUsedBefore time.Time) error {
	var stopper registry.TransitionHook = func(c *registry.ContainerInfo, _ core.ContainerStatus) error {
		// All transitions lock container info.
		// We check container last use time here once again to be sure nobody took it between locks.
		if !c.LastUsed.Before(lastUsedBefore) {
			return fmt.Errorf("someone used the container '%s' before it was scheduled for stopping", c.ID)
		}

		err := s.docker.StopContainer(ctx, c.ID)
		if err != nil {
			return err
		}

		return nil
	}

	return container.ToStopped(stopper, logTransition)
}

func logTransition(c *registry.ContainerInfo, newStatus core.ContainerStatus) error {
	log.Printf("[BG] container '%s' transitioned from '%s' to '%s'", c.ID, c.Status.String(), newStatus.String())
	return nil
}
