package registry

import (
	"fmt"
	"sync"
	"time"

	"github.com/denkoren/mi-labs-test/internal/core"
)

type ContainerInfo struct {
	sync.Mutex
	registry *ContainerRegistry

	core.ContainerInfo
}

func (c *ContainerInfo) UpdateLastUsed() {
	c.Lock()
	c.LastUsed = time.Now()
	c.Unlock()
}

func (c *ContainerInfo) Save() error {
	c.registry.available.set(c)
	c.registry.paramsIndex.set(c)

	// Perform DB update actions here
	return nil
}

var ErrTransitionNotAllowed = fmt.Errorf("transition not allowed")

type TransitionHook func(newStatus core.ContainerStatus) error

var allowedTransitions = map[core.ContainerStatus][]core.ContainerStatus{
	core.ContainerStatusNew:      {core.ContainerStatusStarting, core.ContainerStatusFailed},
	core.ContainerStatusStarting: {core.ContainerStatusReady, core.ContainerStatusFailed},
	core.ContainerStatusReady:    {core.ContainerStatusStopping, core.ContainerStatusFailed},
	core.ContainerStatusStopping: {core.ContainerStatusStopped},
	core.ContainerStatusStopped:  {},
	core.ContainerStatusFailed:   {},
}

func (c *ContainerInfo) transition(status core.ContainerStatus, hooks ...TransitionHook) error {
	c.Lock()
	defer c.Unlock()

	for _, allowedTarget := range allowedTransitions[c.Status] {
		if allowedTarget == status {
			for _, hook := range hooks {
				err := hook(status)
				if err != nil {
					return fmt.Errorf("transition from '%s' to '%s' failed: %w", c.Status.String(), status.String(), err)
				}
			}
			c.Status = status
			c.Updated = time.Now()
			return c.Save()
		}
	}

	return fmt.Errorf("transition from '%s' to '%s' failed: %w", c.Status.String(), status.String(), ErrTransitionNotAllowed)
}

func (c *ContainerInfo) ToStarting(hooks ...TransitionHook) error {
	hooks = append(
		hooks,
		simpleHook(func(_ core.ContainerStatus) { c.Scheduled = time.Now() }),
	)

	return c.transition(core.ContainerStatusStarting, hooks...)
}

func (c *ContainerInfo) ToReady(hooks ...TransitionHook) error {
	hooks = append(
		hooks,
		simpleHook(func(_ core.ContainerStatus) { c.Started = time.Now() }),
	)
	return c.transition(core.ContainerStatusReady, hooks...)
}

func (c *ContainerInfo) ToStopping(hooks ...TransitionHook) error {
	hooks = append(
		hooks,
		simpleHook(func(_ core.ContainerStatus) { c.Stopped = time.Now() }),
	)
	return c.transition(core.ContainerStatusStopping, hooks...)
}

func (c *ContainerInfo) ToStopped() error {
	return c.transition(core.ContainerStatusStopped)
}

func (c *ContainerInfo) ToFailed() error {
	return c.transition(core.ContainerStatusFailed)
}

func simpleHook(f func(ns core.ContainerStatus)) TransitionHook {
	return func(n core.ContainerStatus) error {
		f(n)
		return nil
	}
}

//func convertContainerInfo(container *apipb.Container_Info, r *ContainerRegistry) (*ContainerInfo, error) {
//	c, err := apipb.ConvertContainerInfo(container)
//	if err != nil {
//		return nil, err
//	}
//	return &ContainerInfo{
//		registry: r,
//		ContainerInfo: c,
//	}, err
//}
