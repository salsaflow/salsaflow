package modules

import (
	"github.com/salsita/SalsaFlow/git-trunk/errors"
)

func Bootstrap() *errors.Error {
	return initIssueTracker()
}
