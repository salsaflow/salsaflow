package pkg

import (
	// Stdlib
	"errors"
	"fmt"
	"sort"

	// Internal
	"github.com/salsita/salsaflow/app"
	"github.com/salsita/salsaflow/asciiart"
	"github.com/salsita/salsaflow/errs"
	"github.com/salsita/salsaflow/log"
	"github.com/salsita/salsaflow/prompt"

	// Other
	"github.com/coreos/go-semver/semver"
	"github.com/google/go-github/github"
)

func Upgrade(opts *InstallOptions) error {
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

	// Load GitHub config.
	msg := "Load GitHub config"
	config, err := loadConfig()
	if err != nil {
		return errs.NewError(msg, nil, err)
	}

	// Instantiate a GitHub client.
	msg = "Instantiate a GitHub client"
	client, err := newGitHubClient(config.GitHubToken())
	if err != nil {
		return errs.NewError(msg, nil, err)
	}

	// Fetch the list of available GitHub releases.
	msg = fmt.Sprintf("Fetch GitHub releases for %v/%v", owner, repo)
	log.Run(msg)
	releases, _, err := client.Repositories.ListReleases(owner, repo, nil)
	if err != nil {
		return errs.NewError(msg, nil, err)
	}

	// Sort the releases by version and get the most recent release.
	msg = "Select the most suitable GitHub release"
	var rs releaseSlice
	for _, release := range releases {
		// Skip drafts and pre-releases.
		if *release.Draft || *release.Prerelease {
			continue
		}
		// We expect the tag to be "v" + semver version string.
		version, err := semver.NewVersion((*release.TagName)[1:])
		if err != nil {
			log.Warn(fmt.Sprintf("Tag format invalid for '%v', skipping...", release.TagName))
			continue
		}
		// Append the release to the list of releases.
		rs = append(rs, &githubRelease{
			version:  version,
			resource: &release,
		})
	}
	if rs.Len() == 0 {
		return errs.NewError(msg, nil, errors.New("no suitable GitHub releases found"))
	}

	sort.Sort(rs)
	release := rs[0]

	// Make sure the selected release is more recent than this executable.
	currentVersion, err := semver.NewVersion(app.Version)
	if err != nil {
		panic(err)
	}
	if release.version.String() == app.Version || release.version.LessThan(*currentVersion) {
		log.Log("SalsaFlow is up to date")
		asciiart.PrintThumbsUp()
		fmt.Println()
		return nil
	}

	// Prompt the user to confirm the upgrade.
	msg = "Prompt the user to confirm upgrade"
	fmt.Println()
	confirmed, err := prompt.Confirm(fmt.Sprintf(
		"SalsaFlow version %v is available. Upgrade now?", release.version))
	if err != nil {
		return errs.NewError(msg, nil, err)
	}
	if !confirmed {
		return ErrAborted
	}
	fmt.Println()

	// Proceed to actually install the executables.
	return doInstall(client, owner, repo, release.resource, release.version.String())
}

type githubRelease struct {
	version  *semver.Version
	resource *github.RepositoryRelease
}

type releaseSlice []*githubRelease

func (rs releaseSlice) Len() int {
	return len(rs)
}

func (rs releaseSlice) Less(i, j int) bool {
	return rs[i].version.LessThan(*(rs[j].version))
}

func (rs releaseSlice) Swap(i, j int) {
	rs[i], rs[j] = rs[j], rs[i]
}
