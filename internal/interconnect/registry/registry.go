package registry

import (
	"errors"
	"sync"

	"github.com/denkoren/mi-labs-test/internal/core"
)

var ErrContainerNotExists = errors.New("container does not exist")

type ContainerRegistry struct {
	available   idIndex
	paramsIndex paramsIndex
	stopping    idIndex

	registryLock sync.RWMutex
}

func NewContainerRegistry() (*ContainerRegistry, error) {
	return &ContainerRegistry{
		available:   make(idIndex),
		paramsIndex: make(paramsIndex),
		stopping:    make(idIndex),
	}, nil
}

//func (r *ContainerRegistry) Register(container *apipb.Container_Info) error {
//	r.registryLock.Lock()
//	defer r.registryLock.Unlock()
//
//	if c, _ := r.getByID(container.Id); c != nil {
//		return fmt.Errorf("container with ID '%s' already exist", container.Id)
//	}
//
//	c, err := convertContainerInfo(container, r)
//	if err != nil {
//		return err
//	}
//
//	r.available.set(c)
//	r.paramsIndex.set(c)
//	return nil
//}

// GetByID is thread-safe way to get existing container by its ID
// The container is returned locked, so you should unlock it when you finish operate with it
func (r *ContainerRegistry) GetByID(id string) (*ContainerInfo, error) {
	r.registryLock.RLock()
	defer r.registryLock.Unlock()

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
	defer r.registryLock.Unlock()

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

	err = c.ToStopping()
	if err != nil {
		return err
	}

	r.available.del(id)
	r.paramsIndex.del(c)

	r.stopping.set(c)
	return nil
}

func (r *ContainerRegistry) ToStopped(id string) (*ContainerInfo, error) {
	r.registryLock.Lock()
	defer r.registryLock.Unlock()

	c, err := r.getByID(id)
	if c == nil {
		c = r.stopping[id]
	}
	if c == nil {
		return nil, ErrContainerNotExists
	}

	err = c.ToStopped()
	if err != nil {
		return c, err
	}

	r.available.del(id)
	r.paramsIndex.del(c)
	r.stopping.del(id)

	return c, nil
}
