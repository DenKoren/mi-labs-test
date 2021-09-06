package registry

import (
	"fmt"
	"sync"
	"time"

	"github.com/denkoren/mi-labs-test/internal/core"
)

type ContainerInfo struct {
	sync.Mutex
	registry         *ContainerRegistry
	subscribers      map[int]subscriber
	nextSubscriberID int

	core.ContainerInfo
}

func NewContainerInfo(r *ContainerRegistry, coreInfo core.ContainerInfo) *ContainerInfo {
	return &ContainerInfo{
		registry:         r,
		subscribers:      make(map[int]subscriber, defaultStatusSubscriptionCapacity),
		nextSubscriberID: 0,

		ContainerInfo: coreInfo,
	}
}

func (c *ContainerInfo) UpdateLastUsed() {
	c.Lock()
	c.LastUsed = time.Now()
	c.Unlock()
}

func (c *ContainerInfo) Save() error {
	// Perform DB update actions here
	return nil
}

var ErrTransitionNotAllowed = fmt.Errorf("transition not allowed")

type TransitionHook func(c *ContainerInfo, newStatus core.ContainerStatus) error

var allowedTransitions = map[core.ContainerStatus][]core.ContainerStatus{
	core.ContainerStatusNew: {
		core.ContainerStatusCreated,
		core.ContainerStatusFailed,
	},

	core.ContainerStatusCreated: {
		core.ContainerStatusStarting,
		core.ContainerStatusRunning,
		core.ContainerStatusReady,
	},

	core.ContainerStatusStarting: {
		core.ContainerStatusRunning,
		core.ContainerStatusPaused,
		core.ContainerStatusStopped,
		core.ContainerStatusUnreachable,
	},

	core.ContainerStatusRunning: {
		core.ContainerStatusReady,
		core.ContainerStatusUnreachable,
		core.ContainerStatusPaused,
		core.ContainerStatusStopped,
	},

	core.ContainerStatusReady: {
		core.ContainerStatusRunning,
		core.ContainerStatusUnreachable,
		core.ContainerStatusPaused,
		core.ContainerStatusStopped,
	},

	core.ContainerStatusUnreachable: {
		core.ContainerStatusRunning,
		core.ContainerStatusReady,
		core.ContainerStatusPaused,
		core.ContainerStatusStopped,
	},

	core.ContainerStatusPaused: {
		core.ContainerStatusStarting,
		core.ContainerStatusStopped,
	},

	core.ContainerStatusStopped: {core.ContainerStatusStarting},
	core.ContainerStatusFailed:  {},
}

func (c *ContainerInfo) runHooks(newStatus core.ContainerStatus, hooks ...TransitionHook) error {
	for _, hook := range hooks {
		err := hook(c, newStatus)
		if err != nil {
			return fmt.Errorf("transition from '%s' to '%s' failed: %w", c.Status.String(), newStatus.String(), err)
		}
	}

	return nil
}

func (c *ContainerInfo) transition(newStatus core.ContainerStatus, hooks ...TransitionHook) error {
	c.Lock()
	defer c.Unlock()

	if newStatus == c.Status {
		// Allow transitions from any status to itself
		return c.runHooks(newStatus, hooks...)
	}

	for _, allowedTarget := range allowedTransitions[c.Status] {
		if allowedTarget == newStatus {
			err := c.runHooks(newStatus, hooks...)
			if err != nil {
				return err
			}
			c.Status = newStatus
			c.Updated = time.Now()

			err = c.Save()
			if err != nil {
				return err
			}

			c.notifySubscribers(newStatus)
			return nil
		}
	}

	return fmt.Errorf("transition from '%s' to '%s' failed: %w", c.Status.String(), newStatus.String(), ErrTransitionNotAllowed)
}

func (c *ContainerInfo) ToCreated(hooks ...TransitionHook) error {
	return c.transition(core.ContainerStatusCreated, hooks...)
}

func (c *ContainerInfo) ToStarting(hooks ...TransitionHook) error {
	hooks = append(
		hooks,
		simpleHook(func() { c.Scheduled = time.Now() }),
	)

	return c.transition(core.ContainerStatusStarting, hooks...)
}

func (c *ContainerInfo) ToRunning(hooks ...TransitionHook) error {
	return c.transition(core.ContainerStatusRunning, hooks...)
}

func (c *ContainerInfo) ToReady(hooks ...TransitionHook) error {
	hooks = append(
		hooks,
		simpleHook(func() {
			if c.Status == core.ContainerStatusStarting {
				c.Started = time.Now()
			}
		}),
	)
	return c.transition(core.ContainerStatusReady, hooks...)
}

func (c *ContainerInfo) ToUnreachable(hooks ...TransitionHook) error {
	return c.transition(core.ContainerStatusUnreachable, hooks...)
}

func (c *ContainerInfo) ToPaused(hooks ...TransitionHook) error {
	return c.transition(core.ContainerStatusPaused, hooks...)
}

func (c *ContainerInfo) ToStopped(hooks ...TransitionHook) error {
	hooks = append(
		hooks,
		simpleHook(func() { c.Stopped = time.Now() }),
	)
	return c.transition(core.ContainerStatusStopped, hooks...)
}

func (c *ContainerInfo) ToFailed(hooks ...TransitionHook) error {
	return c.transition(core.ContainerStatusFailed, hooks...)
}

// Hook that does not return error
func simpleHook(f func()) TransitionHook {
	return func(_ *ContainerInfo, _ core.ContainerStatus) error {
		f()
		return nil
	}
}
