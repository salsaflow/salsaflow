package loader

import (
	"fmt"
)

// emptyModuleConfigContainer implements ModuleContainer interface.
type emptyModuleConfigContainer struct {
	configKey  string
	moduleKind ModuleKind
}

// NewEmptyModuleContainer can be used to get a ConfigContainer implementation that does nothing.
// This is handy in case the module doesn't need any configuration. Even if that is true,
// the relevant config spec must return a non-nil ConfigContainer so that the active module ID
// is set to the right module ID. And that is exactly what this function is for.
//
// You can use it like:
//
//     func (spec *configSpec) LocalConfig() loader.ConfigContainer {
//         return loader.NewEmtyModuleConfigContainer(ModuleId, ModuleKind)
//	   }
//
func NewEmptyModuleConfigContainer(configKey string, moduleKind ModuleKind) ConfigContainer {
	return &emptyModuleConfigContainer{
		configKey:  configKey,
		moduleKind: moduleKind,
	}
}

func (container *emptyModuleConfigContainer) ConfigKey() string {
	return container.configKey
}

func (container *emptyModuleConfigContainer) ModuleKind() ModuleKind {
	return container.moduleKind
}

func (container *emptyModuleConfigContainer) PromptUserForConfig() error {
	fmt.Println("Nothing to configure here!")
	return nil
}

func (container *emptyModuleConfigContainer) Marshal() (interface{}, error) {
	return struct{}{}, nil
}

func (container *emptyModuleConfigContainer) Unmarshal(unmarshal func(interface{}) error) error {
	return nil
}
