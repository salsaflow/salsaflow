package github

import (
	// Internal
	"github.com/salsaflow/salsaflow/config/loader"
	"github.com/salsaflow/salsaflow/modules/common"
)

const (
	ModuleId   = "salsaflow.modules.releasenotes.github"
	ModuleKind = loader.ModuleKindReleaseNotes
)

type module struct{}

func NewModule() loader.Module {
	return &module{}
}

func (mod *module) Id() string {
	return ModuleId
}

func (mod *module) Kind() loader.ModuleKind {
	return ModuleKind
}

func (mod *module) ConfigSpec() loader.ModuleConfigSpec {
	return &configSpec{}
}

func (mod *module) NewReleaseNotesManager() (common.ReleaseNotesManager, error) {
	return newReleaseNotesManager()
}
