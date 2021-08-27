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

	Created  time.Time
	Updated  time.Time
	LastUsed time.Time
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

func (c *ContainerInfo) UpdateLastUse() {
	c.LastUsed = time.Now()
}

type ContainerParams struct {
	Seed  string
	Input string
}

type ContainerStatus int

const (
	ContainerStatusNew ContainerStatus = iota
	ContainerStatusStarting
	ContainerStatusReady
	ContainerStatusRunning
	ContainerStatusStopping
	ContainerStatusStopped
)
