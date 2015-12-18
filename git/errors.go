package git

import (
	"errors"
	"fmt"
)

var ErrDirtyRepository = errors.New("the repository is dirty")

type ErrDirtyFile struct {
	RelativePath string
}

func (err *ErrDirtyFile) Error() string {
	if err.RelativePath == "" {
		panic("ErrDirtyFile.relativePath is not set")
	}
	return "file modified but not committed: " + err.RelativePath
}

type ErrRefNotFound struct {
	Ref string
}

func (err *ErrRefNotFound) Error() string {
	return fmt.Sprintf("ref '%v' not found", err.Ref)
}

type ErrRefNotInSync struct {
	Ref string
}

func (err *ErrRefNotInSync) Error() string {
	return fmt.Sprintf("ref '%v' is not up to date", err.Ref)
}
