package pivotaltracker

import (
	// Internal
	"github.com/salsaflow/salsaflow/config/loader"
	"github.com/salsaflow/salsaflow/modules/common"
)

const (
	ModuleId   = "salsaflow.issuetracking.pivotaltracker"
	ModuleKind = loader.ModuleKindIssueTracking
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

func (mod *module) NewIssueTracker() (common.IssueTracker, error) {
	return newIssueTracker()
}
