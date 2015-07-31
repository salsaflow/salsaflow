package pivotaltracker

import (
	// Stdlib
	"fmt"
	"strings"

	// Internal
	"github.com/salsaflow/salsaflow/version"
)

func getReleaseLabel(ver *version.Version) string {
	return fmt.Sprintf("release-%v", ver.BaseString())
}

func isReleaseLabel(label string) bool {
	// Label is always "release-x.y.z"
	if !strings.HasPrefix(label, "release-") {
		return false
	}

	// Skip "release-" and parse the rest as a version string.
	_, err := version.Parse(label[8:])
	return err == nil
}
