package version

import (
	// Stdlib
	"strings"

	// Internal
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
	return Parse(strings.TrimSpace(stdout.String()))
}

// Set runs the set_version script.
func Set(ver *Version) error {
	// Run the set_version script.
	_, err := scripts.Run(scripts.ScriptNameSetVersion, ver.String())
	return err
}
