package pkg

import (
	// Stdlib
	"errors"
	"fmt"
	"sort"

	// Internal
	"github.com/salsaflow/salsaflow/app/metadata"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/prompt"
	"github.com/salsaflow/salsaflow/version"

	// Vendor
	"github.com/google/go-github/github"
)

func Upgrade(opts *InstallOptions) (upgraded bool, err error) {
	// Get GitHub owner and repository names.
	var (
		owner = DefaultGitHubOwner
		repo  = DefaultGitHubRepo
	)
	if opts != nil {
		if opts.GitHubOwner != "" {
			owner = opts.GitHubOwner
		}
		if opts.GitHubRepo != "" {
			repo = opts.GitHubRepo
		}
	}

	// Instantiate a GitHub client.
	task := "Instantiate a GitHub client"
	client, err := newGitHubClient()
	if err != nil {
		return false, errs.NewError(task, err)
	}

	// Fetch the list of available GitHub releases.
	task = fmt.Sprintf("Fetch GitHub releases for %v/%v", owner, repo)
	log.Run(task)
	releases, _, err := client.Repositories.ListReleases(owner, repo, nil)
	if err != nil {
		return false, errs.NewError(task, err)
	}

	// Sort the releases by version and get the most recent release.
	task = "Select the most suitable GitHub release"
	var rs releaseSlice
	for i, release := range releases {
		// Skip drafts and pre-releases.
		if *release.Draft || *release.Prerelease {
			continue
		}
		// We expect the tag to be "v" + semver version string.
		version, err := version.Parse((*release.TagName)[1:])
		if err != nil {
			log.Warn(fmt.Sprintf("Tag format invalid for '%v', skipping...", release.TagName))
			continue
		}
		// Append the release to the list of releases.
		rs = append(rs, &githubRelease{
			version:  version,
			resource: &releases[i],
		})
	}
	if rs.Len() == 0 {
		return false, errs.NewError(task, errors.New("no suitable GitHub releases found"))
	}

	sort.Sort(rs)
	release := rs[len(rs)-1]

	// Make sure the selected release is more recent than this executable.
	currentVersion, err := version.Parse(metadata.Version)
	if err != nil {
		panic(err)
	}
	if release.version.String() == metadata.Version || release.version.LT(currentVersion.Version) {
		return false, nil
	}

	// Prompt the user to confirm the upgrade.
	task = "Prompt the user to confirm upgrade"
	fmt.Println()
	confirmed, err := prompt.Confirm(fmt.Sprintf(
		"SalsaFlow version %v is available. Upgrade now?", release.version), true)
	if err != nil {
		return false, errs.NewError(task, err)
	}
	if !confirmed {
		return false, ErrAborted
	}
	fmt.Println()

	// Proceed to actually install the executables.
	err = doInstall(client, owner, repo, release.resource.Assets, release.version.String(), "")
	return err == nil, err
}

type githubRelease struct {
	version  *version.Version
	resource *github.RepositoryRelease
}

type releaseSlice []*githubRelease

func (rs releaseSlice) Len() int {
	return len(rs)
}

func (rs releaseSlice) Less(i, j int) bool {
	return rs[i].version.LT(rs[j].version.Version)
}

func (rs releaseSlice) Swap(i, j int) {
	rs[i], rs[j] = rs[j], rs[i]
}
