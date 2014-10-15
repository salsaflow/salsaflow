package git

import "errors"

var ErrDirtyRepository = errors.New("the repository is dirty")

type ErrDirtyFile struct {
	relativePath string
}

func (err *ErrDirtyFile) Error() string {
	if err.relativePath == "" {
		panic("ErrDirtyFile.relativePath is not set")
	}
	return "file modified but not committed: " + err.relativePath
}
