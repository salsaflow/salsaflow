package pivotaltracker

import (
	// Stdlib
	"fmt"

	// Internal
	"github.com/salsaflow/salsaflow/version"
)

func getReleaseLabel(ver *version.Version) string {
	return fmt.Sprintf("release-%v", ver.BaseString())
}
