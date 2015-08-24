package github

import (
	// Stdlib
	"bytes"
	"fmt"

	// Internal
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/github"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/modules/common"
	"github.com/salsaflow/salsaflow/releases/notes"

	// Vendor
	gh "github.com/google/go-github/github"
	"github.com/toqueteos/webbrowser"
)

const Id = "github"

const LocalConfigTemplate = "# no additional config required"

type moduleFactory struct{}

func NewFactory() common.ReleaseNotesManagerFactory {
	return &moduleFactory{}
}

func (factory *moduleFactory) LocalConfigTemplate() string {
	return LocalConfigTemplate
}

func (factory *moduleFactory) NewReleaseNotesManager() (common.ReleaseNotesManager, error) {
	return &releaseNotesManager{}, nil
}

type releaseNotesManager struct{}

func (rnm *releaseNotesManager) PostReleaseNotes(
	releaseNotes *common.ReleaseNotes,
) (action.Action, error) {

	// Get the GitHub owner and repository from the upstream URL.
	owner, repo, err := github.ParseUpstreamURL()
	if err != nil {
		return nil, err
	}

	// Instantiate the API client.
	config, err := github.LoadConfig()
	if err != nil {
		return nil, err
	}
	client := github.NewClient(config.ApiToken())

	// Format the release notes.
	task := "Format the release notes"
	body := bytes.NewBufferString(`
## Summary ##

**PLEASE FILL IN THE RELEASE SUMMARY**
`)
	encoder, err := notes.NewEncoder(notes.EncodingMarkdown, body)
	if err != nil {
		return nil, errs.NewError(task, err)
	}
	if err := encoder.Encode(releaseNotes, nil); err != nil {
		return nil, errs.NewError(task, err)
	}
	bodyString := body.String()

	// Create GitHub release for the given version.
	tag := releaseNotes.Version.ReleaseTagString()
	releaseTask := fmt.Sprintf("Create GitHub release for tag '%v'", tag)
	log.Run(releaseTask)
	release, _, err := client.Repositories.CreateRelease(owner, repo, &gh.RepositoryRelease{
		TagName: gh.String(tag),
		Name:    gh.String("Release " + releaseNotes.Version.BaseString()),
		Body:    &bodyString,
		Draft:   gh.Bool(true),
	})
	if err != nil {
		return nil, err
	}

	// Delete the GitHub release on rollback.
	rollback := func() error {
		log.Rollback(releaseTask)
		task := fmt.Sprintf("Delete GitHub release for tag '%v'", tag)
		_, err := client.Repositories.DeleteRelease(owner, repo, *release.ID)
		if err != nil {
			return errs.NewError(task, err)
		}
		return nil
	}

	// Open the release in the browser so that the user can fill in the details.
	task = "Open the release notes in the browser"
	if err := webbrowser.Open(*release.HTMLURL); err != nil {
		if ex := rollback(); ex != nil {
			errs.Log(ex)
		}
		return nil, errs.NewError(task, err)
	}

	// Return the rollback function.
	return action.ActionFunc(rollback), nil
}
