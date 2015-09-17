package github

import (
	// Stdlib
	"fmt"
	"net/url"
	"regexp"
	"strings"

	// Internal
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/git"
)

// ParseUpstreamURL parses the URL of the git upstream being used by SalsaFlow
// and returns the given GitHub owner and repository.
func ParseUpstreamURL() (owner, repo string, err error) {
	// Load the Git config.
	gitConfig, err := git.LoadConfig()
	if err != nil {
		return "", "", err
	}
	remoteName := gitConfig.RemoteName

	// Get the upstream URL.
	task := fmt.Sprintf("Get URL for git remote '%v'", remoteName)
	remoteURL, err := git.GetConfigString(fmt.Sprintf("remote.%v.url", remoteName))
	if err != nil {
		return "", "", errs.NewError(task, err)
	}

	// Parse it and return the result.
	return parseUpstreamURL(remoteURL)
}

func parseUpstreamURL(remoteURL string) (owner, repo string, err error) {
	// Parse the upstream URL to get the owner and repo name.
	task := "Parse the upstream repository URL"

	defer func() {
		// Strip trailing .git if present.
		if strings.HasSuffix(repo, ".git") {
			repo = repo[:len(repo)-len(".git")]
		}
	}()

	// Try to parse the URL as an SSH URL first.
	owner, repo, ok := tryParseUpstreamAsSSH(remoteURL)
	if ok {
		return owner, repo, nil
	}

	// Try to parse the URL as a regular URL.
	owner, repo, ok = tryParseUpstreamAsURL(remoteURL)
	if ok {
		return owner, repo, nil
	}

	// No success, return an error.
	err = fmt.Errorf("failed to parse git remote URL: %v", remoteURL)
	return "", "", errs.NewError(task, err)
}

// tryParseUpstreamAsSSH tries to parse the address as an SSH address,
// e.g. git@github.com:owner/repo.git
func tryParseUpstreamAsSSH(remoteURL string) (owner, repo string, ok bool) {
	re := regexp.MustCompile("^git@[^:]+:([^/]+)/(.+)$")
	match := re.FindStringSubmatch(remoteURL)
	if len(match) != 0 {
		owner, repo := match[1], match[2]
		return owner, repo, true
	}

	return "", "", false
}

// tryParseUpstreamAsURL tries to parse the address as a regular URL,
// e.g. https://github.com/owner/repo
func tryParseUpstreamAsURL(remoteURL string) (owner, repo string, ok bool) {
	u, err := url.Parse(remoteURL)
	if err != nil {
		return "", "", false
	}

	switch u.Scheme {
	case "ssh":
		fallthrough
	case "https":
		re := regexp.MustCompile("^/([^/]+)/(.+)$")
		match := re.FindStringSubmatch(u.Path)
		if len(match) != 0 {
			owner, repo := match[1], match[2]
			return owner, repo, true
		}
	}

	return "", "", false
}
