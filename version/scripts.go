package version

import (
	// Stdlib
	"strings"

	// Internal
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/scripts"
)

// Get runs the get_version script.
func Get() (*Version, error) {
	// Run the get_version script.
	stdout, err := scripts.Run(scripts.ScriptNameGetVersion)
	if err != nil {
		return nil, err
	}

	// Parse the output and return the version.
	task := "Parse the version string"
	ver, err := Parse(strings.TrimSpace(stdout.String()))
	if err != nil {
		return nil, errs.NewError(task, err, nil)
	}
	return ver, nil
}

// Set runs the set_version script.
func Set(ver *Version) error {
	// Run the set_version script.
	_, err := scripts.Run(scripts.ScriptNameSetVersion, ver.String())
	return err
}
