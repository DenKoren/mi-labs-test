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
	Seed  string
	Input string
}

type ContainerStatus int

const (
	ContainerStatusNew      ContainerStatus = iota // Was just added to in-memory registry.
	ContainerStatusStarting                        // Was scheduled for start in management system.
	ContainerStatusReady                           // Was started, is healthy and ready to handle requests.
	ContainerStatusStopping                        // Is stopping. It can't handle requests but still exist in in-memory registry.
	ContainerStatusStopped                         // Was stopped and already removed from in-memory registry.
	ContainerStatusFailed                          // Failed to start or does not respond to requests.
)

func (s ContainerStatus) IsAvailable() bool {
	return s < ContainerStatusStopping
}
