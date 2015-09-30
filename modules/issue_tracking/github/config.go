package github

import (
	// Stdlib
	"fmt"
	"strings"

	// Internal
	"github.com/salsaflow/salsaflow/config/loader"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/github"
	"github.com/salsaflow/salsaflow/prompt"
)

// Configuration ===============================================================

type moduleConfig struct {
	// GitHub repository.
	GitHubOwner      string
	GitHubRepository string

	// GitHub API authentication.
	UserToken string

	// Story label.
	StoryLabel string

	// State labels.
	ApprovedLabel         string
	BeingImplementedLabel string
	ImplementedLabel      string
	ReviewedLabel         string
	SkipReviewLabel       string
	PassedTestingLabel    string
	FailedTestingLabel    string
	SkipTestingLabel      string
	StagedLabel           string
	RejectedLabel         string
	SkipCheckLabels       []string
}

func loadConfig() (*moduleConfig, error) {
	task := fmt.Sprintf("Load config for module '%v'", ModuleId)

	// Load the config.
	spec := newConfigSpec()
	if err := loader.LoadConfig(spec); err != nil {
		return nil, errs.NewError(task, err)
	}

	// Parse the main repo upstream URL.
	owner, repo, err := github.ParseUpstreamURL()
	if err != nil {
		return nil, errs.NewError(task, err)
	}

	// Assemble the config object.
	var (
		local  = spec.local
		global = spec.global
	)
	return &moduleConfig{
		GitHubOwner:           owner,
		GitHubRepository:      repo,
		UserToken:             global.UserToken,
		StoryLabel:            local.StoryLabel,
		ApprovedLabel:         local.StateLabels.ApprovedLabel,
		BeingImplementedLabel: local.StateLabels.BeingImplementedLabel,
		ImplementedLabel:      local.StateLabels.ImplementedLabel,
		ReviewedLabel:         local.StateLabels.ReviewedLabel,
		SkipReviewLabel:       local.StateLabels.SkipReviewLabel,
		PassedTestingLabel:    local.StateLabels.PassedTestingLabel,
		FailedTestingLabel:    local.StateLabels.FailedTestingLabel,
		SkipTestingLabel:      local.StateLabels.SkipTestingLabel,
		StagedLabel:           local.StateLabels.StagedLabel,
		RejectedLabel:         local.StateLabels.RejectedLabel,
		SkipCheckLabels:       local.SkipCheckLabels,
	}, nil
}

// Configuration spec ----------------------------------------------------------

type configSpec struct {
	global *GlobalConfig
	local  *LocalConfig
}

func newConfigSpec() *configSpec {
	return &configSpec{}
}

// ConfigKey is a part of loader.ConfigSpec
func (spec *configSpec) ConfigKey() string {
	return ModuleId
}

// ModuleKind is a part of loader.ModuleConfigSpec
func (spec *configSpec) ModuleKind() loader.ModuleKind {
	return loader.ModuleKindIssueTracking
}

// GlobalConfig is a part of loader.ConfigSpec
func (spec *configSpec) GlobalConfig() loader.ConfigContainer {
	spec.global = &GlobalConfig{}
	return spec.global
}

// LocalConfig is a part of loader.ConfigSpec
func (spec *configSpec) LocalConfig() loader.ConfigContainer {
	spec.local = &LocalConfig{spec: spec}
	return spec.local
}

// Global configuration --------------------------------------------------------

// GlobalConfig implements loader.ConfigContainer
type GlobalConfig struct {
	UserToken string `prompt:"GitHub token to be used when calling GitHub API" secret:"true" json:"token"`
}

// PromptUserForConfig is a part of loader.ConfigContainer
func (global *GlobalConfig) PromptUserForConfig() error {
	var c GlobalConfig
	if err := prompt.Dialog(&c, "Insert the"); err != nil {
		return err
	}

	*global = c
	return nil
}

// Local configuration ---------------------------------------------------------

const (
	DefaultStoryLabel            = "story"
	DefaultApprovedLabel         = "approved"
	DefaultBeingImplementedLabel = "being implemented"
	DefaultImplementedLabel      = "implemented"
	DefaultReviewedLabel         = "reviewed"
	DefaultSkipReviewLabel       = "no review"
	DefaultPassedTestingLabel    = "qa+"
	DefaultFailedTestingLabel    = "qa-"
	DefaultSkipTestingLabel      = "no qa"
	DefaultStagedLabel           = "staged"
	DefaultRejectedLabel         = "client rejected"
)

var DefaultSkipCheckLabels = []string{"dupe", "wontfix"}

// LocalConfig implements loader.ConfigContainer interface.
type LocalConfig struct {
	spec *configSpec

	StoryLabel string `json:"story_label"`

	StateLabels struct {
		ApprovedLabel         string `json:"approved"`
		BeingImplementedLabel string `json:"being_implemented"`
		ImplementedLabel      string `json:"implemented"`
		ReviewedLabel         string `json:"reviewed"`
		SkipReviewLabel       string `json:"skip_review"`
		PassedTestingLabel    string `json:"passed_testing"`
		FailedTestingLabel    string `json:"failed_testing"`
		SkipTestingLabel      string `json:"skip_testing"`
		StagedLabel           string `json:"staged_for_acceptance"`
		RejectedLabel         string `json:"client_rejected"`
	} `json:"state_labels"`

	SkipCheckLabels []string `json:"skip_release_check_labels"`
}

// PromptUserForConfig is a part of loader.ConfigContainer interface.
func (local *LocalConfig) PromptUserForConfig() error {
	c := LocalConfig{spec: local.spec}

	// Prompt for the labels.
	var err error
	promptForLabel := func(dst *string, labelName, defaultValue string) {
		if err != nil {
			return
		}
		question := fmt.Sprintf("%v label", labelName)
		var label string
		label, err = prompt.PromptDefault(question, defaultValue)
		if err == nil {
			*dst = label
		}
	}

	promptForLabel(&c.StoryLabel, "Story", DefaultStoryLabel)

	promptForLabel(&c.StateLabels.ApprovedLabel, "Approved", DefaultApprovedLabel)
	promptForLabel(
		&c.StateLabels.BeingImplementedLabel, "Being implemented", DefaultBeingImplementedLabel)
	promptForLabel(&c.StateLabels.ImplementedLabel, "Implemented", DefaultImplementedLabel)
	promptForLabel(&c.StateLabels.ReviewedLabel, "Reviewed", DefaultReviewedLabel)
	promptForLabel(&c.StateLabels.SkipReviewLabel, "Skip review", DefaultSkipReviewLabel)
	promptForLabel(&c.StateLabels.PassedTestingLabel, "Passed testing", DefaultPassedTestingLabel)
	promptForLabel(&c.StateLabels.FailedTestingLabel, "Failed testing", DefaultFailedTestingLabel)
	promptForLabel(&c.StateLabels.SkipTestingLabel, "Skip testing", DefaultSkipTestingLabel)
	promptForLabel(&c.StateLabels.StagedLabel, "Staged", DefaultStagedLabel)
	promptForLabel(&c.StateLabels.RejectedLabel, "Client rejected", DefaultRejectedLabel)
	if err != nil {
		return err
	}

	// Prompt for the release skip check labels.
	skipCheckLabelsString, err := prompt.Prompt(fmt.Sprintf(
		"Skip check labels, comma-separated (%v always included): ",
		strings.Join(DefaultSkipCheckLabels, ", ")))
	if err != nil {
		if err != prompt.ErrCanceled {
			return err
		}
	}

	// Append the new labels to the default ones.
	// Make sure there are no duplicates and empty strings.
	var (
		insertedLabels = strings.Split(skipCheckLabelsString, ",")
		lenDefault     = len(DefaultSkipCheckLabels)
		lenInserted    = len(insertedLabels)
	)

	// Save a few allocations.
	skipCheckLabels := make([]string, lenDefault, lenDefault+lenInserted)
	copy(skipCheckLabels, DefaultSkipCheckLabels)

LabelLoop:
	for _, insertedLabel := range insertedLabels {
		// Trim spaces.
		insertedLabel = strings.TrimSpace(insertedLabel)

		// Skip empty strings.
		if insertedLabel == "" {
			continue
		}

		// Make sure there are no duplicates.
		for _, existingLabel := range skipCheckLabels {
			if insertedLabel == existingLabel {
				continue LabelLoop
			}
		}

		// Append the label.
		skipCheckLabels = append(skipCheckLabels, insertedLabel)
	}
	c.SkipCheckLabels = skipCheckLabels

	// Success!
	*local = c
	return nil
}
