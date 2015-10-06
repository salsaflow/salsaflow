package loader

import (
	"fmt"
)

// emptyModuleConfigContainer implements ModuleContainer interface.
// It is used internally when a nil ConfigContainer is returned so that
// the logic can be simpler by not having to treat this scenario separately.
type emptyModuleConfigContainer struct {
	configKey  string
	moduleKind ModuleKind
}

func newEmptyModuleConfigContainer(configKey string, moduleKind ModuleKind) ConfigContainer {
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
