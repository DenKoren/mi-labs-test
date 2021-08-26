package docker

import (
	"log"

	"github.com/denkoren/mi-labs-test/internal/util"
	apipb "github.com/denkoren/mi-labs-test/proto/v1"
)

type ContainerManager struct {
}

func NewContainerManager() (*ContainerManager, error) {
	return &ContainerManager{}, nil
}

func (*ContainerManager) StartContainer(params *apipb.Container_Params) (string, error) {
	log.Printf("Starting container: seed: %s, input: %s", params.Seed, params.Input)
	return util.RandString(10), nil
}

func (*ContainerManager) GetContainerInfo(id string) (*apipb.Container_Info, error) {
	return &apipb.Container_Info{
		Id: id,
	}, nil
}

func (*ContainerManager) StopContainer(id string) error {
	log.Printf("Stopping container: id: %s", id)
	return nil
}
