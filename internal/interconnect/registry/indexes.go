package registry

type idIndex map[string]*ContainerInfo

func (s idIndex) set(c *ContainerInfo) {
	s[c.ID] = c
}

func (s idIndex) del(id string) {
	delete(s, id)
}

type seedIndex map[string]*ContainerInfo

func (s seedIndex) set(c *ContainerInfo) {
	s[c.Params.Seed] = c
}

func (s seedIndex) del(c *ContainerInfo) {
	delete(s, c.Params.Seed)
}
