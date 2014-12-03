package scripts

import (
	// Stdlib
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	// Internal
	"github.com/salsaflow/salsaflow/config"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/git/gitutil"
)

const ScriptDirname = "scripts"

const (
	ScriptNameGetVersion = "get_version"
	ScriptNameSetVersion = "set_version"
)

type ErrNotFound struct {
	scriptName string
}

func (err *ErrNotFound) Error() string {
	return fmt.Sprintf("custom SalsaFlow script '%v' not found", err.scriptName)
}

func (err *ErrNotFound) ScriptName() string {
	return err.scriptName
}

func Run(name string, args ...string) (stdout *bytes.Buffer, err error) {
	// Get the repository root.
	root, err := gitutil.RepositoryRootAbsolutePath()
	if err != nil {
		return nil, err
	}

	// Get the local config directory path.
	scripts := filepath.Join(root, config.LocalConfigDirname, ScriptDirname)

	// Get the script absolute path based on the platform.
	scriptPath := filepath.Join(scripts, name)

	// Try just the script name as it is.
	if _, err := os.Stat(scriptPath); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	// Didn't find the file, try to add platform suffix,
	// which can be either "windows" or "unix".
	if runtime.GOARCH == "windows" {
		scriptPath += "_windows"
	} else {
		scriptPath += "_unix"
	}
	if _, err := os.Stat(scriptPath); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		return nil, &ErrNotFound{name}
	}

	// Run the given script in the repository root.
	task := fmt.Sprintf("Run the %v script", name)
	var (
		sout bytes.Buffer
		serr bytes.Buffer
	)
	cmd := exec.Command(scriptPath, args...)
	cmd.Stdout = &sout
	cmd.Stderr = &serr
	cmd.Dir = root
	if err := cmd.Run(); err != nil {
		return nil, errs.NewError(task, err, &serr)
	}
	return &sout, nil
}
