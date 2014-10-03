package modules

import (
	"github.com/salsita/salsaflow/errors"
)

func Bootstrap() *errors.Error {
	return initIssueTracker()
}
