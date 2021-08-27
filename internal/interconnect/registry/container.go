package registry

import (
	"sync"

	"github.com/denkoren/mi-labs-test/internal/core"
)

type ContainerInfo struct {
	mutex sync.Mutex
	registry *ContainerRegistry

	core.ContainerInfo
}

func (c *ContainerInfo) Save() {
	c.registry.containers.set(c)
	c.registry.index.set(c)

	// Perform DB update actions here
}

func (c *ContainerInfo) Release() {
	c.mutex.Unlock()
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
