package noop

import (
	// Internal
	"github.com/salsaflow/salsaflow/config/loader"
)

// Configuration spec ----------------------------------------------------------

type configSpec struct{}

func newConfigSpec() *configSpec {
	return &configSpec{}
}

// ConfigKey is a part of loader.ConfigSpec
func (spec *configSpec) ConfigKey() string {
	return ModuleId
}

// ModuleKind is a part of loader.ModuleConfigSpec
func (spec *configSpec) ModuleKind() loader.ModuleKind {
	return ModuleKind
}

// GlobalConfig is a part of loader.ConfigSpec
func (spec *configSpec) GlobalConfig() loader.ConfigContainer {
	return nil
}

// LocalConfig is a part of loader.ConfigSpec
func (spec *configSpec) LocalConfig() loader.ConfigContainer {
	return nil
}
