package sprintly

import (
	// Stdlib
	"fmt"

	// Internal
	"github.com/salsaflow/salsaflow/version"
)

func getItemReleaseTag(ver *version.Version) string {
	return fmt.Sprintf("release-%v", ver)
}
