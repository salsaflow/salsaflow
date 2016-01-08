package pkg

import (
	// Stdlib
	"fmt"

	// Internal
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/prompt"

	// Other
	"github.com/google/go-github/github"
)

type InstallOptions struct {
	GitHubOwner     string
	GitHubRepo      string
	TargetDirectory string
}

func Install(version string, opts *InstallOptions) error {
	// Get GitHub owner and repository names.
	var (
		owner     = DefaultGitHubOwner
		repo      = DefaultGitHubRepo
		targetDir string
	)
	if opts != nil {
		if opts.GitHubOwner != "" {
			owner = opts.GitHubOwner
		}
		if opts.GitHubRepo != "" {
			repo = opts.GitHubRepo
		}
		targetDir = opts.TargetDirectory
	}

	// Instantiate a GitHub client.
	task := "Instantiate a GitHub client"
	client, err := newGitHubClient()
	if err != nil {
		return errs.NewError(task, err)
	}

	// Fetch the list of available GitHub releases.
	task = fmt.Sprintf("Fetch GitHub releases for %v/%v", owner, repo)
	log.Run(task)
	releases, err := listReleases(client, owner, repo)
	if err != nil {
		return errs.NewError(task, err)
	}

	// Get the release matching the chosen version string.
	task = "Get the release metadata"
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
		return errs.NewError(task, fmt.Errorf("SalsaFlow version %v not found", version))
	}

	// Prompt the user to confirm the the installation.
	task = "Prompt the user to confirm the installation"
	fmt.Println()
	confirmed, err := prompt.Confirm(fmt.Sprintf(
		"SalsaFlow version %v is about to be installed. Shall we proceed?", version), true)
	if err != nil {
		return errs.NewError(task, err)
	}
	if !confirmed {
		return ErrAborted
	}
	fmt.Println()

	// Proceed to actually install the executables.
	return doInstall(client, owner, repo, release.Assets, version, targetDir)
}
