package v1

import "github.com/denkoren/mi-labs-test/internal/core"

func ConvertContainerStatus(status Container_Status) (core.ContainerStatus, error) {
	switch status {
	case Container_NEW:
		return core.ContainerStatusNew, nil
	case Container_STARTING:
		return core.ContainerStatusStarting, nil
	case Container_READY:
		return core.ContainerStatusReady, nil
	case Container_STOPPING:
		return core.ContainerStatusStopping, nil
	case Container_STOPPED:
		return core.ContainerStatusStopped, nil
	case Container_FAILED:
		return core.ContainerStatusFailed, nil
	default:
		return core.ContainerStatus(status), core.ErrUnknownContainerStatus
	}
}

func ConvertContainerParams(params *Container_Params) core.ContainerParams {
	return core.ContainerParams{
		Seed:  params.GetSeed(),
		Input: params.GetInput(),
	}
}

func ConvertContainerInfo(container *Container_Info) (core.ContainerInfo, error) {
	status, err := ConvertContainerStatus(container.GetStatus())
	if err != nil {
		return core.ContainerInfo{}, err
	}

	c := core.NewContainerInfo(
		container.GetId(),
		container.GetAddr(),
		ConvertContainerParams(container.GetParams()),
	)
	c.Status = status
	return c, err
}
