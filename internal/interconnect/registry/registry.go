package registry

import (
	"errors"
	"fmt"
	"sync"

	"github.com/denkoren/mi-labs-test/internal/core"
)

const defaultContainerRegistryCapacity = 10

var ErrContainerNotExists = errors.New("container does not exist")

type ContainerRegistry struct {
	available   idIndex
	paramsIndex paramsIndex
	stopping    idIndex
	failed      idIndex

	registryLock sync.RWMutex
}

func NewContainerRegistry() (*ContainerRegistry, error) {
	return &ContainerRegistry{
		available:   make(idIndex, defaultContainerRegistryCapacity),
		paramsIndex: make(paramsIndex, defaultContainerRegistryCapacity),
		stopping:    make(idIndex, defaultContainerRegistryCapacity),
		failed:      make(idIndex, defaultContainerRegistryCapacity),
	}, nil
}

func (r *ContainerRegistry) Register(c *ContainerInfo) error {
	r.registryLock.Lock()
	defer r.registryLock.Unlock()

	if c, _ := r.getByID(c.ID); c != nil {
		return fmt.Errorf("container with ID '%s' already exist", c.ID)
	}

	r.available.set(c)
	r.paramsIndex.set(c)
	return nil
}

// GetByID is thread-safe way to get existing container by its ID
// The container is returned locked, so you should unlock it when you finish operate with it
func (r *ContainerRegistry) GetByID(id string) (*ContainerInfo, error) {
	r.registryLock.RLock()
	defer r.registryLock.RUnlock()

	c, err := r.getByID(id)
	if err != nil {
		return nil, err
	}

	c.UpdateLastUsed()
	return c, nil
}

// getByID returns existing container by its ID
// is NOT thread safe
func (r *ContainerRegistry) getByID(id string) (*ContainerInfo, error) {
	c, exists := r.available[id]
	if !exists {
		return nil, ErrContainerNotExists
	}

	return c, nil
}

// GetByParams is a thread-safe way to get existing container by its parameters.
// The container is returned locked, so you should unlock it when you finish operate with it
func (r *ContainerRegistry) GetByParams(params core.ContainerParams) (*ContainerInfo, error) {
	r.registryLock.RLock()
	defer r.registryLock.RUnlock()

	c, err := r.getByParams(params)
	if err != nil {
		return nil, err
	}

	c.UpdateLastUsed()
	return c, nil
}

// getByParams returns existing container by its parameters
// is NOT thread safe
func (r *ContainerRegistry) getByParams(params core.ContainerParams) (*ContainerInfo, error) {
	inIndex, exists := r.paramsIndex[params.Seed]
	if !exists {
		return nil, ErrContainerNotExists
	}

	c, exists := inIndex[params.Input]
	if !exists {
		return nil, ErrContainerNotExists
	}

	return c, nil
}

func (r *ContainerRegistry) ExistingOrNewByParams(params core.ContainerParams) (*ContainerInfo, error) {
	r.registryLock.Lock()
	defer r.registryLock.Unlock()

	c, err := r.getByParams(params)
	if err == nil {
		c.UpdateLastUsed()
		return c, nil
	}

	if err != ErrContainerNotExists {
		return nil, err
	}

	c = &ContainerInfo{
		registry: r,

		ContainerInfo: core.NewContainerInfo(
			"",
			"",
			params,
		),
	}
	r.paramsIndex.set(c)

	return c, nil
}

func (r *ContainerRegistry) ToStopping(id string) error {
	r.registryLock.Lock()
	defer r.registryLock.Unlock()

	c, err := r.getByID(id)
	if err != nil {
		return err
	}

	err = c.ToStopping(
		func(_ core.ContainerStatus) error {
			r.available.del(id)
			r.paramsIndex.del(c)

			r.stopping.set(c)
			return nil
		},
	)
	return err
}

func (r *ContainerRegistry) ToStopped(id string) (*ContainerInfo, error) {
	r.registryLock.Lock()
	defer r.registryLock.Unlock()

	c, _ := r.getByID(id)
	if c == nil {
		c = r.stopping[id]
	}
	if c == nil {
		return nil, ErrContainerNotExists
	}

	return c, c.ToStopped(
		func(_ core.ContainerStatus) error {
			r.available.del(id)
			r.paramsIndex.del(c)
			r.stopping.del(id)
			return nil
		},
	)
}
