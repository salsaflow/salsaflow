package bootstrapCmd

import (
	// Stdlib
	"sort"

	// Internal
	"github.com/salsaflow/salsaflow/config/loader"
)

func groupModulesByKind(modules []loader.Module) [][]loader.Module {
	m := make(map[loader.ModuleKind]*[]loader.Module)
	for _, module := range modules {
		kind := module.Kind()
		listPtr, ok := m[kind]
		if !ok {
			list := []loader.Module{module}
			m[kind] = &list
			continue
		}
		*listPtr = append(*listPtr, module)
	}

	groups := make([][]loader.Module, 0, len(m))
	for _, v := range m {
		ms := *v
		sort.Sort(commonModules(ms))
		groups = append(groups, ms)
	}
	return groups
}

// Implement sort.Interface to sort by Id alphabetically.
type commonModules []loader.Module

func (ms commonModules) Len() int {
	return len(ms)
}

func (ms commonModules) Less(i, j int) bool {
	return ms[i].Id() < ms[j].Id()
}

func (ms commonModules) Swap(i, j int) {
	ms[i], ms[j] = ms[j], ms[i]
}
