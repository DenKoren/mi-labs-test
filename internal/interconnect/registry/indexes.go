package registry

type idIndex map[string]*ContainerInfo

func (s idIndex) set(c *ContainerInfo) {
	s[c.ID] = c
}

func (s idIndex) del(id string) {
	delete(s, id)
}

type inputIndex map[string]*ContainerInfo
type paramsIndex map[string]inputIndex

func (s paramsIndex) set(c *ContainerInfo) {
	iIndex, exists := s[c.Params.Seed]
	if !exists {
		iIndex = make(inputIndex, defaultContainerRegistryCapacity)
		s[c.Params.Seed] = iIndex
	}

	iIndex[c.Params.Input] = c
}

func (s paramsIndex) del(c *ContainerInfo) {
	index, exists := s[c.Params.Seed]
	if !exists {
		return
	}

	delete(index, c.Params.Input)
	if len(index) != 0 {
		return
	}

	delete(s, c.Params.Seed)
}
