package git

import (
	"errors"
	"fmt"
)

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

type ErrRefNotFound struct {
	ref string
}

func (err *ErrRefNotFound) Error() string {
	return fmt.Sprintf("ref '%v' not found", err.ref)
}

func (err *ErrRefNotFound) Ref() string {
	return err.ref
}

type ErrRefNotInSync struct {
	ref string
}

func (err *ErrRefNotInSync) Error() string {
	return fmt.Sprintf("ref '%v' is not up to date", err.ref)
}

func (err *ErrRefNotInSync) Ref() string {
	return err.ref
}
