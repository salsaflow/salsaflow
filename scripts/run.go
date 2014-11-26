package scripts

import (
	// Stdlib
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"

	// Internal
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/git/gitutil"
)

const (
	ScriptNameGetVersion = "get_version"
	ScriptNameSetVersion = "set_version"
)

func Run(name string, args ...string) (stdout *bytes.Buffer, err error) {
	// Get the scripts config section.
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	// Get the repository root.
	root, err := gitutil.RepositoryRootAbsolutePath()
	if err != nil {
		return nil, err
	}

	// Get the script relative path.
	var relativePath string
	switch name {
	case ScriptNameGetVersion:
		relativePath = config.GetVersionScriptRelativePath()
	case ScriptNameSetVersion:
		relativePath = config.SetVersionScriptRelativePath()
	default:
		panic("unknown script name: " + name)
	}

	// Run the given script in the repository root.
	task := fmt.Sprintf("Run the %v script", name)
	scriptAbsPath := filepath.Join(root, relativePath)
	var (
		sout bytes.Buffer
		serr bytes.Buffer
	)
	cmd := exec.Command(scriptAbsPath, args...)
	cmd.Stdout = &sout
	cmd.Stderr = &serr
	cmd.Dir = root
	if err := cmd.Run(); err != nil {
		return nil, errs.NewError(task, err, &serr)
	}
	return &sout, nil
}
