package background

import (
	"context"
	"fmt"
	"log"
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
	go s.stopInactiveContainers(ctx, wg)

	wg.Add(1)
	go s.cleanupStoppedContainers(ctx, wg)

	wg.Wait()
	return nil
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
				log.Printf("failed to load old containers list")
				continue
			}

			log.Printf("Detected '%d' old containers", len(containers))
			for _, container := range containers {
				err = s.scheduleContainerStop(ctx, container.ID, lastUsedBefore)
				if err != nil {
					log.Printf("background service failed to stop container '%s': %v", container.ID, err)
					continue
				}

				log.Printf("background service scheduled container '%s' stop", container.ID)
			}

		case <-ctx.Done():
			log.Printf("background task 'stopInactiveContainers' context done: %v", ctx.Err())
			return
		}
	}
}

func (s *Background) scheduleContainerStop(ctx context.Context, containerID string, lastUsedBefore time.Time) error {
	var stopper registry.TransitionHook = func(c *registry.ContainerInfo, _ core.ContainerStatus) error {
		// All transitions lock container info.
		// We check container last use time here once again to be sure nobody took it between locks.
		if !c.LastUsed.Before(lastUsedBefore) {
			return fmt.Errorf("someone used the container '%s' before it was scheduled for stopping", c.ID)
		}

		return s.docker.StopContainer(ctx, c.ID)
	}

	return s.registry.ToStopping(containerID, stopper)
}

func (s *Background) cleanupStoppedContainers(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	ticker := time.NewTicker(s.config.ContainersCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			containers, err := s.registry.StoppingContainers()
			if err != nil {
				log.Printf("failed to load stopping containers list")
				continue
			}

			log.Printf("Loaded '%d' stopping containers", len(containers))
			for _, container := range containers {
				err = s.markContainerStopped(ctx, container.ID)
				if err != nil {
					log.Printf("background service failed to mark container as '%s' as stopped: %v", container.ID, err)
					continue
				}

				log.Printf("background service stopped container '%s'", container.ID)
			}

		case <-ctx.Done():
			log.Printf("background task 'stopInactiveContainers' context done: %v", ctx.Err())
			return
		}
	}
}

func (s *Background) markContainerStopped(ctx context.Context, containerID string) error {
	stopped, err := s.docker.IsContainerStopped(ctx, containerID)
	if err != nil {
		return err
	}
	if !stopped {
		return fmt.Errorf("container '%s' is not stopped yet", containerID)
	}

	_, err = s.registry.ToStopped(containerID)
	return err
}
