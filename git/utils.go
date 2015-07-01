package git

import (
	// Stdlib
	"fmt"
	"net/url"
	"regexp"

	// Internal
	"github.com/salsaflow/salsaflow/errs"
)

// ParseUpstreamURL parses the URL of the git upstream being used by SalsaFlow
// and returns the given GitHub owner and repository.
func ParseUpstreamURL() (owner, repo string, err error) {
	// Load the Git config.
	gitConfig, err := LoadConfig()
	if err != nil {
		return "", "", err
	}
	remoteName := gitConfig.RemoteName()

	// Get the upstream URL.
	task := fmt.Sprintf("Get URL for git remote '%v'", remoteName)
	remoteURL, err := GetConfigString(fmt.Sprintf("remote.%v.url", remoteName))
	if err != nil {
		return "", "", errs.NewError(task, err)
	}

	// Parse the upstream URL to get the owner and repo name.
	task = "Parse the upstream repository URL"
	u, err := url.Parse(remoteURL)
	if err != nil {
		return "", "", errs.NewError(task, err)
	}

	var match []string
	if u.Scheme == "https" {
		// Handle HTTPS.
		re := regexp.MustCompile("/([^/]+)/(.+)")
		match = re.FindStringSubmatch(u.Path)
	} else {
		// Handle SSH.
		re := regexp.MustCompile("git@github.com:([^/]+)/(.+)[.]git")
		match = re.FindStringSubmatch(remoteURL)
	}
	if len(match) != 3 {
		err := fmt.Errorf("failed to parse git remote URL: %v", remoteURL)
		return "", "", errs.NewError(task, err)
	}
	return match[1], match[2], nil
}
