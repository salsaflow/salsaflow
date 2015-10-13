package gitutil

import (
	// Stdlib
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	// Internal
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/shell"
)

func Run(args ...string) (stdout *bytes.Buffer, err error) {
	argsList := make([]string, 1, 1+len(args))
	argsList[0] = "--no-pager"
	argsList = append(argsList, args...)

	task := fmt.Sprintf("Run git with args = %#v", args)
	log.V(log.Debug).Log(task)
	stdout, stderr, err := shell.Run("git", argsList...)
	if err != nil {
		return nil, errs.NewErrorWithHint(task, err, stderr.String())
	}
	return stdout, nil
}

func RunCommand(command string, args ...string) (stdout *bytes.Buffer, err error) {
	argsList := make([]string, 2, 2+len(args))
	argsList[0], argsList[1] = "--no-pager", command
	argsList = append(argsList, args...)

	task := fmt.Sprintf("Run 'git %v' with args = %#v", command, args)
	log.V(log.Debug).Log(task)
	stdout, stderr, err := shell.Run("git", argsList...)
	if err != nil {
		return nil, errs.NewErrorWithHint(task, err, stderr.String())
	}
	return stdout, nil
}

func RepositoryRootAbsolutePath() (path string, err error) {
	task := "Get the repository root absolute path"
	stdout, err := Run("rev-parse", "--show-toplevel")
	if err != nil {
		return "", errs.NewError(task, err)
	}
	return string(bytes.TrimSpace(stdout.Bytes())), nil
}

// RelativePath returns the relative path from the current working directory to the file
// specified by the relative path from the repository root.
//
// This is useful for some other Git commands, particularly git status.
func RelativePath(pathFromRoot string) (relativePath string, err error) {
	task := fmt.Sprintf("Get the relative for '%v'", pathFromRoot)

	root, err := RepositoryRootAbsolutePath()
	if err != nil {
		return "", errs.NewError(task, err)
	}
	absolutePath := filepath.Join(root, pathFromRoot)

	cwd, err := os.Getwd()
	if err != nil {
		return "", errs.NewError(task, err)
	}

	relativePath, err = filepath.Rel(cwd, absolutePath)
	if err != nil {
		return "", errs.NewError(task, err)
	}

	return relativePath, nil
}

func ShowFileByBranch(file, branch string) (content *bytes.Buffer, err error) {
	return RunCommand("show", fmt.Sprintf("%v:%v", branch, file))
}

func CurrentBranch() (branch string, err error) {
	stdout, err := Run("symbolic-ref", "--short", "HEAD")
	if err != nil {
		return "", err
	}

	return string(bytes.TrimSpace(stdout.Bytes())), nil
}

func CurrentUpstreamBranch() (upstreamBranch string, err error) {
	stdout, err := Run("symbolic-ref", "-q", "HEAD")
	if err != nil {
		return "", err
	}
	branch = string(bytes.TrimSpace(stdout.Bytes()))

	stdout, err = Run("for-each-ref", "--format=%(upstream:short)", branch)
	if err != nil {
		return "", err
	}

	return string(bytes.TrimSpace(stdout.Bytes())), nil
}
