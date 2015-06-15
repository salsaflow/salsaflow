package scripts

import (
	// Stdlib
	"bytes"
	"fmt"

	// Internal
	"github.com/salsaflow/salsaflow/errs"
)

const (
	ScriptNameGetVersion = "get_version"
	ScriptNameSetVersion = "set_version"
)

func Run(scriptName string, args ...string) (stdout *bytes.Buffer, err error) {
	task := fmt.Sprintf("Run the %v script", scriptName)

	// Get the command for the given script name and args.
	cmd, err := Command(scriptName, args...)
	if err != nil {
		return nil, errs.NewError(task, err)
	}

	// Run the given script in the repository root.
	var (
		sout bytes.Buffer
		serr bytes.Buffer
	)
	cmd.Stdout = &sout
	cmd.Stderr = &serr
	if err := cmd.Run(); err != nil {
		return nil, errs.NewErrorWithHint(task, err, serr.String())
	}
	return &sout, nil
}
