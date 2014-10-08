package git

import "errors"

var ErrDirtyRepository = errors.New("the repository is dirty")

type ErrDirtyFile struct {
	relativePath string
}

func (err *ErrDirtyFile) Error() string {
	return "file modified but not committed: " + err.relativePath
}
