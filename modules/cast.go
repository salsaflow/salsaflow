package modules

import (
	"github.com/salsaflow/salsaflow/config/loader"
	"github.com/salsaflow/salsaflow/modules/common"
)

func AsIssueTrackingModule(module loader.Module) (common.IssueTrackingModule, error) {
	mod, ok := module.(common.IssueTrackingModule)
	if !ok {
		return nil, &ErrInvalidModule{module.Id(), loader.ModuleKindIssueTracking}
	}
	return mod, nil
}

func AsCodeReviewModule(module loader.Module) (common.CodeReviewModule, error) {
	mod, ok := module.(common.CodeReviewModule)
	if !ok {
		return nil, &ErrInvalidModule{module.Id(), loader.ModuleKindCodeReview}
	}
	return mod, nil
}

func AsReleaseNotesModule(module loader.Module) (common.ReleaseNotesModule, error) {
	mod, ok := module.(common.ReleaseNotesModule)
	if !ok {
		return nil, &ErrInvalidModule{module.Id(), loader.ModuleKindReleaseNotes}
	}
	return mod, nil
}
