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
	idIndex   idIndex
	seedIndex seedIndex
	stopping  idIndex
	failed      idIndex

	indexesLock sync.RWMutex
}

func NewContainerRegistry() (*ContainerRegistry, error) {
	return &ContainerRegistry{
		idIndex:   make(idIndex, defaultContainerRegistryCapacity),
		seedIndex: make(seedIndex, defaultContainerRegistryCapacity),
		stopping:  make(idIndex, defaultContainerRegistryCapacity),
		failed:    make(idIndex, defaultContainerRegistryCapacity),
	}, nil
}

func (r *ContainerRegistry) Register(c *ContainerInfo) error {
	r.indexesLock.Lock()
	defer r.indexesLock.Unlock()

	if c, _ := r.getByID(c.ID); c != nil {
		return fmt.Errorf("container with ID '%s' already exist", c.ID)
	}

	r.idIndex.set(c)
	r.seedIndex.set(c)
	return nil
}

// GetByID is thread-safe way to get existing container by its ID
// The container is returned locked, so you should unlock it when you finish operate with it
func (r *ContainerRegistry) GetByID(id string) (*ContainerInfo, error) {
	r.indexesLock.RLock()
	defer r.indexesLock.RUnlock()

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
	c, exists := r.idIndex[id]
	if !exists {
		return nil, ErrContainerNotExists
	}

	return c, nil
}

// GetByParams is a thread-safe way to get existing container by its parameters.
// The container is returned locked, so you should unlock it when you finish operate with it
func (r *ContainerRegistry) GetByParams(params core.ContainerParams) (*ContainerInfo, error) {
	r.indexesLock.RLock()
	defer r.indexesLock.RUnlock()

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
	r.indexesLock.Lock()
	defer r.indexesLock.Unlock()

	c, err := r.getByParams(params)
	if err == nil {
		c.UpdateLastUsed()
		return c, nil
	}

	if err != ErrContainerNotExists {
		return nil, err
	}

	c = NewContainerInfo(
		r,
		core.NewContainerInfo(
			"",
			"",
			params,
		),
	)

	r.seedIndex.set(c)
	go r.registerContainerID(c)

	return c, nil
}

// registerContainerID stores container in ID index once it is created.
// We don't know container ID when we register new record in registry.
// We also can't use public Register method to avoid deadlocks between
// parallel API requests.
// That is why we subscribe the status change event and wait container started
// to update internal ID index.
func (r *ContainerRegistry) registerContainerID(c *ContainerInfo) {
	subscription := c.subscribe()
	defer subscription.Unsubscribe()

	if c.Status.WasCreated() {
		r.indexesLock.Lock()
		r.idIndex.set(c)
		r.indexesLock.Unlock()
		return
	}

	for {
		<-subscription.C
		if c.Status.WasCreated() {
			r.indexesLock.Lock()
			r.idIndex.set(c)
			r.indexesLock.Unlock()
			return
		}
	}
}

func (r *ContainerRegistry) Delete(id string) error {
	r.indexesLock.Lock()
	defer r.indexesLock.Unlock()

	c, err := r.getByID(id)
	if err != nil {
		return err
	}

	r.idIndex.del(id)
	r.seedIndex.del(c)
	return nil
}

func (r *ContainerRegistry) ActiveContainers() ([]*ContainerInfo, error) {
	r.indexesLock.RLock()
	defer r.indexesLock.RUnlock()

	result := make([]*ContainerInfo, 0, defaultContainerRegistryCapacity)
	for _, container := range r.idIndex {
		if container.Status.IsActive() {
			result = append(result, container)
		}
	}

	return result, nil
}

func (r *ContainerRegistry) OldContainers(lastUsedBefore time.Time) ([]*ContainerInfo, error) {
	r.indexesLock.RLock()
	defer r.indexesLock.RUnlock()

	result := make([]*ContainerInfo, 0, defaultContainerRegistryCapacity)
	for _, container := range r.idIndex {
		if container.Status.IsActive() && container.LastUsed.Before(lastUsedBefore) {
			result = append(result, container)
		}
	}

	return result, nil
}
