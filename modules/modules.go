package modules

import (
	"github.com/salsita/salsaflow/errs"
)

func Bootstrap() *errs.Error {
	return initIssueTracker()
}
