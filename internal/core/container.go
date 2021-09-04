//go:generate stringer -type=ContainerStatus -trimprefix=ContainerStatus -output status_string.go

package core

import (
	"errors"
	"time"
)

var ErrUnknownContainerStatus = errors.New("unknown container status")

type ContainerInfo struct {
	ID     string
	Addr   string
	Params ContainerParams
	Status ContainerStatus

	Created   time.Time
	Scheduled time.Time
	Started   time.Time
	Stopped   time.Time
	Updated   time.Time
	LastUsed  time.Time
}

func NewContainerInfo(id string, addr string, params ContainerParams) ContainerInfo {
	now := time.Now()
	return ContainerInfo{
		ID:       id,
		Addr:     addr,
		Params:   params,
		Status:   ContainerStatusNew,
		Created:  now,
		Updated:  now,
		LastUsed: now,
	}
}

type ContainerParams struct {
	Seed string
}

type ContainerStatus int

const (
	ContainerStatusNew     ContainerStatus = iota // Was just added to in-memory registry.
	ContainerStatusCreated                        // Was created in Docker service

	// Active statuses
	ContainerStatusStarting    // Was scheduled for start in management system.
	ContainerStatusRunning     // Was started, but health-check indicates container is not ready.
	ContainerStatusReady       // Was started, is healthy and ready to handle requests.
	ContainerStatusUnreachable // Was started, but is unreachable by network.

	// Inactive statuses
	ContainerStatusPaused  // Was paused.
	ContainerStatusStopped // Was stopped and already removed from in-memory registry.
	ContainerStatusFailed  // Container failed to start due to system error (e.g. docker error).
)

func (i ContainerStatus) WasCreated() bool {
	return i >= ContainerStatusCreated
}

// IsActive becomes true when docker container is running
func (i ContainerStatus) IsActive() bool {
	return ContainerStatusStarting <= i && i <= ContainerStatusUnreachable
}

// IsStartable becomes true when docker container is not running, but we can start it
func (i ContainerStatus) IsStartable() bool {
	return i == ContainerStatusStopped ||
		i == ContainerStatusPaused ||
		i == ContainerStatusCreated
}
