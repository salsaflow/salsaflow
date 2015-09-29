package pivotaltracker

import (
	// Stdlib
	"fmt"
	"sort"
	"strings"

	// Internal
	"github.com/salsaflow/salsaflow/config/loader"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/prompt"

	// Vendor
	"gopkg.in/salsita/go-pivotaltracker.v1/v5/pivotal"
)

// Configuration ===============================================================

type moduleConfig struct {
	ProjectId        int
	ComponentLabel   string
	PointMeLabel     string
	ReviewedLabel    string
	SkipReviewLabel  string
	TestedLabel      string
	SkipTestingLabel string
	SkipCheckLabels  []string
	UserToken        string
}

func loadConfig() (*moduleConfig, error) {
	// Load the config.
	spec := newConfigSpec()
	if err := loader.LoadConfig(spec); err != nil {
		return nil, err
	}

	// Assemble the config object.
	var (
		local  = spec.local
		global = spec.global
	)
	return &moduleConfig{
		ProjectId:        local.ProjectId,
		ComponentLabel:   *local.ComponentLabel,
		PointMeLabel:     local.Labels.PointMeLabel,
		ReviewedLabel:    local.Labels.ReviewedLabel,
		SkipReviewLabel:  local.Labels.SkipReviewLabel,
		TestedLabel:      local.Labels.TestedLabel,
		SkipTestingLabel: local.Labels.SkipTestingLabel,
		SkipCheckLabels:  local.Labels.SkipCheckLabels,
		UserToken:        global.UserToken,
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
	UserToken string `prompt:"personal Pivotal Tracker token" secret:"true" json:"token"`
}

// PromptUserForConfig is a part of loader.ConfigContainer
func (global *GlobalConfig) PromptUserForConfig() error {
	var c GlobalConfig
	if err := prompt.Dialog(&c, "Insert your"); err != nil {
		return err
	}

	*global = c
	return nil
}

// Local configuration ---------------------------------------------------------

const (
	DefaultPointMeLabel     = "point me"
	DefaultReviewedLabel    = "reviewed"
	DefaultSkipReviewLabel  = "no review"
	DefaultTestedLabel      = "qa+"
	DefaultSkipTestingLabel = "no qa"
)

var DefaultSkipCheckLabels = []string{"dupe", "wontfix"}

// LocalConfig implements loader.ConfigContainer interface.
type LocalConfig struct {
	spec *configSpec

	ProjectId      int     `json:"project_id"`
	ComponentLabel *string `json:"component_label"`
	Labels         struct {
		PointMeLabel     string   `json:"point_me"`
		ReviewedLabel    string   `json:"reviewed"`
		SkipReviewLabel  string   `json:"skip_review"`
		TestedLabel      string   `json:"tested"`
		SkipTestingLabel string   `json:"skip_testing"`
		SkipCheckLabels  []string `json:"skip_release_check_labels"`
	} `json:"workflow_labels"`
}

// PromptUserForConfig is a part of loader.ConfigContainer interface.
func (local *LocalConfig) PromptUserForConfig() error {
	c := LocalConfig{spec: local.spec}

	// Prompt for the project ID.
	task := "Fetch available Pivotal Tracker projects"
	log.Run(task)

	client := pivotal.NewClient(local.spec.global.UserToken)

	projects, err := client.Projects.List()
	if err != nil {
		return errs.NewError(task, err)
	}
	sort.Sort(ptProjects(projects))

	fmt.Println()
	fmt.Println("Available Pivotal Tracker projects:")
	fmt.Println()
	for i, project := range projects {
		fmt.Printf("  [%v] %v\n", i+1, project.Name)
	}
	fmt.Println()
	fmt.Println("Choose the project to associate this repository with.")
	index, err := prompt.PromptIndex("Project number: ", 1, len(projects))
	if err != nil {
		if err == prompt.ErrCanceled {
			prompt.PanicCancel()
		}

		return err
	}
	fmt.Println()

	c.ProjectId = projects[index-1].Id

	// Prompt for the labels.
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

	var componentLabel string
	promptForLabel(&componentLabel, "Component", "")
	c.ComponentLabel = &componentLabel

	promptForLabel(&c.Labels.PointMeLabel, "Point me", DefaultPointMeLabel)
	promptForLabel(&c.Labels.ReviewedLabel, "Reviewed", DefaultReviewedLabel)
	promptForLabel(&c.Labels.SkipReviewLabel, "Skip review", DefaultSkipReviewLabel)
	promptForLabel(&c.Labels.TestedLabel, "Testing passed", DefaultTestedLabel)
	promptForLabel(&c.Labels.SkipTestingLabel, "Skip testing", DefaultSkipTestingLabel)
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
	c.Labels.SkipCheckLabels = skipCheckLabels

	// Success!
	*local = c
	return nil
}

// Implement sort.Interface to sort projects alphabetically.
type ptProjects []*pivotal.Project

func (ps ptProjects) Len() int {
	return len(ps)
}

func (ps ptProjects) Less(i, j int) bool {
	return ps[i].Name < ps[j].Name
}

func (ps ptProjects) Swap(i, j int) {
	ps[i], ps[j] = ps[j], ps[i]
}
