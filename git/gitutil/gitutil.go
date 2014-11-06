package gitutil

import (
	// Stdlib
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	// Internal
	"github.com/salsita/salsaflow/errs"
	"github.com/salsita/salsaflow/shell"
)

func Run(args ...string) (stdout, stderr *bytes.Buffer, err error) {
	argsList := make([]string, 2, 2+len(args))
	argsList[0], argsList[1] = "git", "--no-pager"
	argsList = append(argsList, args...)
	return shell.Run(argsList...)
}

func RunCommand(command string, args ...string) (stdout, stderr *bytes.Buffer, err error) {
	argsList := make([]string, 3, 3+len(args))
	argsList[0], argsList[1], argsList[2] = "git", "--no-pager", command
	argsList = append(argsList, args...)
	return shell.Run(argsList...)
}

func RepositoryRootAbsolutePath() (path string, stderr *bytes.Buffer, err error) {
	stdout, stderr, err := Run("rev-parse", "--show-toplevel")
	if err != nil {
		return
	}

	path = string(bytes.TrimSpace(stdout.Bytes()))
	return
}

// RelativePath returns the relative path from the current working directory to the file
// specified by the relative path from the repository root.
//
// This is useful for some other Git commands, particularly git status.
func RelativePath(pathFromRoot string) (relativePath string, stderr *bytes.Buffer, err error) {
	root, stderr, err := RepositoryRootAbsolutePath()
	if err != nil {
		return "", stderr, err
	}
	absolutePath := filepath.Join(root, pathFromRoot)

	cwd, err := os.Getwd()
	if err != nil {
		return "", nil, err
	}

	relativePath, err = filepath.Rel(cwd, absolutePath)
	if err != nil {
		return "", nil, err
	}

	return relativePath, nil, nil
}

func ShowFileByBranch(file, branch string) (content *bytes.Buffer, err error) {
	var (
		object = fmt.Sprintf("%v:%v", branch, file)
		task   = fmt.Sprintf("Run 'git show %v'", object)
	)
	stdout, stderr, err := RunCommand("show", object)
	if err != nil {
		return nil, errs.NewError(task, err, stderr)
	}
	return stdout, nil
}
