package pivotaltracker

import (
	// Stdlib
	"bufio"
	"bytes"
	"regexp"
	"strconv"
	"strings"

	// Internal
	"github.com/salsita/SalsaFlow/git-trunk/config"
	"github.com/salsita/SalsaFlow/git-trunk/git"
)

var (
	localRefMatcher  = "^refs/heads/story/.+/[0-9]+$"
	remoteRefMatcher = "^refs/remotes/" + config.OriginName + "/story/.+/[0-9]+$"
)

func ListGitStoryRefs() (localRefs, remoteRefs []string, stderr *bytes.Buffer, err error) {
	stdout, stderr, err := git.Git("show-ref")
	if err != nil {
		return
	}

	var (
		local  []string
		remote []string

		localMatcher  = regexp.MustCompile(localRefMatcher)
		remoteMatcher = regexp.MustCompile(remoteRefMatcher)
	)
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, " ")
		ref := parts[1]

		switch {
		case localMatcher.MatchString(ref):
			local = append(local, ref)
		case remoteMatcher.MatchString(ref):
			remote = append(remote, ref)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, nil, err
	}

	return local, remote, nil, nil
}

func RefToStoryId(ref string) (storyId int, err error) {
	matcher := regexp.MustCompile("story/.+/([0-9]+)$")
	parts := matcher.FindStringSubmatch(ref)
	if len(parts) != 2 {
		return 0, &ErrNotStoryBranch{ref}
	}

	storyId, _ = strconv.Atoi(parts[1])
	return
}

type ErrNotStoryBranch struct {
	ref string
}

func (err *ErrNotStoryBranch) Error() string {
	return "not a story reference: " + err.ref
}
