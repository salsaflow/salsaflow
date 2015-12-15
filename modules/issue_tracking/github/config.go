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

	// Story labels.
	StoryLabels []string

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
		StoryLabels:           local.StoryLabels,
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
	return ModuleKind
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

var DefaultStoryLabels = []string{"enhancement", "bug"}

const (
	DefaultApprovedLabel         = "approved"
	DefaultBeingImplementedLabel = "being implemented"
	DefaultImplementedLabel      = "implemented"
	DefaultReviewedLabel         = "reviewed"
	DefaultSkipReviewLabel       = "no review"
	DefaultPassedTestingLabel    = "qa+"
	DefaultFailedTestingLabel    = "qa-"
	DefaultSkipTestingLabel      = "no qa"
	DefaultStagedLabel           = "staged"
	DefaultRejectedLabel         = "rejected"
)

var ImplicitSkipCheckLabels = []string{"duplicate", "invalid"}

// LocalConfig implements loader.ConfigContainer interface.
type LocalConfig struct {
	spec *configSpec

	StoryLabels []string `json:"story_labels"`

	StateLabels struct {
		ApprovedLabel         string `prompt:"'approved' label" default:"approved" json:"approved"`
		BeingImplementedLabel string `prompt:"'being implemented' label" default:"being implemented" json:"being_implemented"`
		ImplementedLabel      string `prompt:"'implemented' label" default:"implemented" json:"implemented"`
		ReviewedLabel         string `prompt:"'reviewed' label" default:"reviewed" json:"reviewed"`
		SkipReviewLabel       string `prompt:"'no review' label" default:"no review" json:"skip_review"`
		PassedTestingLabel    string `prompt:"'passed testing' label" default:"qa+" json:"passed_testing"`
		FailedTestingLabel    string `prompt:"'failed testing' label" default:"qa-" json:"failed_testing"`
		SkipTestingLabel      string `prompt:"'skip testing' label" default:"no qa" json:"skip_testing"`
		StagedLabel           string `prompt:"'staged' label" default:"staged" json:"staged_for_acceptance"`
		RejectedLabel         string `prompt:"'rejected' label" default:"rejected" json:"client_rejected"`
	} `json:"state_labels"`

	SkipCheckLabels []string `json:"skip_release_check_labels"`
}

// PromptUserForConfig is a part of loader.ConfigContainer interface.
func (local *LocalConfig) PromptUserForConfig() error {
	c := LocalConfig{spec: local.spec}

	// Prompt for the state labels.
	if err := prompt.Dialog(&c, "Insert the"); err != nil {
		return err
	}

	// Prompt for the story labels.
	storyLabels, err := promptForLabelList("Insert the story labels", DefaultStoryLabels, nil)
	fmt.Println()
	if err != nil {
		return err
	}
	c.StoryLabels = storyLabels

	// Prompt for the release skip check labels.
	skipCheckLabels, err := promptForLabelList(
		"Insert the skip release check labels", nil, ImplicitSkipCheckLabels)
	if err != nil {
		return err
	}
	c.SkipCheckLabels = skipCheckLabels

	// Success!
	*local = c
	return nil
}

func promptForLabelList(msg string, defaultItems, implicitItems []string) ([]string, error) {
	var (
		lenDefault  = len(defaultItems)
		lenImplicit = len(implicitItems)
	)

	// Prompt for the value.
	fmt.Printf("%v, comma-separated.\n", msg)
	if lenDefault != 0 {
		fmt.Printf("  (default values: %v)\n", strings.Join(defaultItems, ", "))
	}
	if lenImplicit != 0 {
		fmt.Printf("  (always included: %v)\n", strings.Join(implicitItems, ", "))
	}
	fmt.Println()
	input, err := prompt.Prompt("Your choice: ")
	if err != nil {
		if err == prompt.ErrCanceled {
			return append(defaultItems, implicitItems...), nil
		}
		return nil, err
	}

	// Append the new labels to the default ones.
	// Make sure there are no duplicates and empty strings.
	var (
		insertedLabels = strings.Split(input, ",")
		lenInserted    = len(insertedLabels)
	)

	// Save a few allocations.
	labels := make([]string, lenImplicit, lenImplicit+lenInserted)
	copy(labels, implicitItems)

LabelLoop:
	for _, insertedLabel := range insertedLabels {
		// Trim spaces.
		insertedLabel = strings.TrimSpace(insertedLabel)

		// Skip empty strings.
		if insertedLabel == "" {
			continue
		}

		// Make sure there are no duplicates.
		for _, existingLabel := range labels {
			if insertedLabel == existingLabel {
				continue LabelLoop
			}
		}

		// Append the label.
		labels = append(labels, insertedLabel)
	}

	return labels, nil
}
