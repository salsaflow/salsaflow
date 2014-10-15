package modules

import (
	"github.com/salsita/salsaflow/errs"
)

func Bootstrap() *errs.Error {
	if err := initIssueTracker(); err != nil {
		return err
	}
	return initCodeReviewTool()
}
