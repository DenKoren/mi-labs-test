package registry

type idIndex map[string]*ContainerInfo

func (s idIndex) set(c *ContainerInfo) {
	s[c.ID] = c
}

type inputIndex map[string]*ContainerInfo
type seedIndex map[string]inputIndex

func (s seedIndex) set(c *ContainerInfo) {
	s[c.Params.Seed][c.Params.Input] = c
}
