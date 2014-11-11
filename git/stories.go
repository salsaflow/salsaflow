package git

import (
	"bufio"
	"regexp"
	"strings"
)

const OriginName = "origin"

var (
	localRefMatcher  = "^refs/heads/story/.+/.+$"
	remoteRefMatcher = "^refs/remotes/" + OriginName + "/story/.+/.+$"
)

func ListStoryRefs() (localRefs, remoteRefs []string, err error) {
	stdout, err := Run("show-ref")
	if err != nil {
		return nil, nil, err
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
		return nil, nil, err
	}

	return local, remote, nil
}

func RefToStoryId(ref string) (storyId string, err error) {
	matcher := regexp.MustCompile("story/.+/(.+)$")
	parts := matcher.FindStringSubmatch(ref)
	if len(parts) != 2 {
		return "", &ErrNotStoryBranch{ref}
	}

	return parts[1], nil
}

type ErrNotStoryBranch struct {
	ref string
}

func (err *ErrNotStoryBranch) Error() string {
	return "not a story reference: " + err.ref
}
