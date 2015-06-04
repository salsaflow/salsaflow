package version

// Versions implement sort.Interface
type Versions []*Version

func (vs Versions) Len() int {
	return len(vs)
}

func (vs Versions) Less(i, j int) bool {
	return vs[i].LT(vs[j].Version)
}

func (vs Versions) Swap(i, j int) {
	vs[i], vs[j] = vs[j], vs[i]
}
