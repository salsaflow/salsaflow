package modules

import (
	// Internal
	"github.com/salsaflow/salsaflow/config"
	"github.com/salsaflow/salsaflow/config/loader"
	"github.com/salsaflow/salsaflow/modules/common"
)

var (
	issueTracker        common.IssueTracker
	codeReviewTool      common.CodeReviewTool
	releaseNotesManager common.ReleaseNotesManager
)

func AvailableModules() []loader.Module {
	modules := make([]loader.Module, len(registeredModules))
	copy(modules, registeredModules)
	return modules
}

func GetIssueTracker() (common.IssueTracker, error) {
	if issueTracker != nil {
		return issueTracker, nil
	}

	module, err := loadActiveModule(loader.ModuleKindIssueTracking)
	if err != nil {
		return nil, err
	}

	trackerModule, err := AsIssueTrackingModule(module)
	if err != nil {
		return nil, err
	}

	implementation, err := trackerModule.NewIssueTracker()
	if err != nil {
		return nil, err
	}

	issueTracker = implementation
	return issueTracker, nil
}

func GetCodeReviewTool() (common.CodeReviewTool, error) {
	if codeReviewTool != nil {
		return codeReviewTool, nil
	}

	module, err := loadActiveModule(loader.ModuleKindCodeReview)
	if err != nil {
		return nil, err
	}

	reviewModule, err := AsCodeReviewModule(module)
	if err != nil {
		return nil, err
	}

	implementation, err := reviewModule.NewCodeReviewTool()
	if err != nil {
		return nil, err
	}

	codeReviewTool = implementation
	return codeReviewTool, nil
}

func GetReleaseNotesManager() (common.ReleaseNotesManager, error) {
	if releaseNotesManager != nil {
		return releaseNotesManager, nil
	}

	module, err := loadActiveModule(loader.ModuleKindReleaseNotes)
	if err != nil {
		return nil, err
	}

	notesModule, err := AsReleaseNotesModule(module)
	if err != nil {
		return nil, err
	}

	implementation, err := notesModule.NewReleaseNotesManager()
	if err != nil {
		return nil, err
	}

	releaseNotesManager = implementation
	return releaseNotesManager, nil
}

func loadActiveModule(kind loader.ModuleKind) (loader.Module, error) {
	// Load local configuration.
	localConfig, err := config.ReadLocalConfig()
	if err != nil {
		return nil, err
	}

	// Get the module matching the module kind.
	activeModuleId := loader.ActiveModule(localConfig, kind)
	if activeModuleId == "" {
		return nil, &ErrModuleNotSet{kind}
	}

	// Find the module among the registered modules.
	for _, module := range registeredModules {
		if module.Id() == activeModuleId {
			return module, nil
		}
	}
	return nil, &ErrModuleNotFound{activeModuleId}
}
