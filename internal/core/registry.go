package core

import (
	"sync"

	apipb "github.com/denkoren/mi-labs-test/proto/api/v1"
)

type inputIndex map[string]*apipb.Container_Info
type seedIndex map[string]inputIndex

type ContainerRegistry struct {
	containers map[string]*apipb.Container_Info
	index seedIndex

	registryLock sync.RWMutex

}

