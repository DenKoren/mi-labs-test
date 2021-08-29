package registry

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/denkoren/mi-labs-test/internal/core"
)

const defaultContainerRegistryCapacity = 10

var ErrContainerNotExists = errors.New("container does not exist")

type ContainerRegistry struct {
	available idIndex
	seedIndex seedIndex
	stopping  idIndex
	failed      idIndex

	registryLock sync.RWMutex
}

func NewContainerRegistry() (*ContainerRegistry, error) {
	return &ContainerRegistry{
		available: make(idIndex, defaultContainerRegistryCapacity),
		seedIndex: make(seedIndex, defaultContainerRegistryCapacity),
		stopping:  make(idIndex, defaultContainerRegistryCapacity),
		failed:    make(idIndex, defaultContainerRegistryCapacity),
	}, nil
}

func (r *ContainerRegistry) Register(c *ContainerInfo) error {
	r.registryLock.Lock()
	defer r.registryLock.Unlock()

	if c, _ := r.getByID(c.ID); c != nil {
		return fmt.Errorf("container with ID '%s' already exist", c.ID)
	}

	r.available.set(c)
	r.seedIndex.set(c)
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
	c, exists := r.seedIndex[params.Seed]
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
	r.seedIndex.set(c)

	return c, nil
}

func (r *ContainerRegistry) ToStopping(id string, hooks ...TransitionHook) error {
	r.registryLock.Lock()
	defer r.registryLock.Unlock()

	c, err := r.getByID(id)
	if err != nil {
		return err
	}

	hooks = append(
		hooks,
		simpleHook(func() {
			r.available.del(id)
			r.seedIndex.del(c)

			r.stopping.set(c)
		}),
	)

	err = c.ToStopping(hooks...)
	return err
}

func (r *ContainerRegistry) ToStopped(id string, hooks ...TransitionHook) (*ContainerInfo, error) {
	r.registryLock.Lock()
	defer r.registryLock.Unlock()

	c, _ := r.getByID(id)
	if c == nil {
		c = r.stopping[id]
	}
	if c == nil {
		return nil, ErrContainerNotExists
	}

	hooks = append(
		hooks,
		simpleHook(func() {
			r.available.del(id)
			r.seedIndex.del(c)
			r.stopping.del(id)
		}),
	)
	return c, c.ToStopped(hooks...)
}

func (r *ContainerRegistry) OldContainers(lastUsedBefore time.Time) ([]*ContainerInfo, error) {
	r.registryLock.RLock()
	defer r.registryLock.RUnlock()

	result := make([]*ContainerInfo, 0, defaultContainerRegistryCapacity)
	for _, container := range r.available {
		if container.LastUsed.Before(lastUsedBefore) {
			result = append(result, container)
		}
	}

	return result, nil
}

func (r *ContainerRegistry) StoppingContainers() ([]*ContainerInfo, error) {
	r.registryLock.RLock()
	defer r.registryLock.RUnlock()

	result := make([]*ContainerInfo, 0, len(r.stopping))
	for _, container := range r.stopping {
		result = append(result, container)
	}

	return result, nil
}
