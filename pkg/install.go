package pkg

import (
	// Stdlib
	"fmt"

	// Internal
	"github.com/salsita/salsaflow/errs"
	"github.com/salsita/salsaflow/log"
	"github.com/salsita/salsaflow/prompt"

	// Other
	"github.com/google/go-github/github"
)

type InstallOptions struct {
	GitHubOwner string
	GitHubRepo  string
}

func Install(version string, opts *InstallOptions) error {
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

	// Get the release matching the chosen version string.
	msg = "Get the release metadata"
	var (
		release *github.RepositoryRelease
		tagName = "v" + version
	)
	for _, r := range releases {
		if *r.TagName == tagName {
			release = &r
			break
		}
	}
	if release == nil {
		return errs.NewError(msg, nil, fmt.Errorf("SalsaFlow version %v not found", version))
	}

	// Prompt the user to confirm the the installation.
	msg = "Prompt the user to confirm the installation"
	fmt.Println()
	confirmed, err := prompt.Confirm(fmt.Sprintf(
		"SalsaFlow version %v is about to be installed. Shall we proceed?", version))
	if err != nil {
		return errs.NewError(msg, nil, err)
	}
	if !confirmed {
		return ErrAborted
	}
	fmt.Println()

	// Proceed to actually install the executables.
	return doInstall(client, owner, repo, release, version)
}
