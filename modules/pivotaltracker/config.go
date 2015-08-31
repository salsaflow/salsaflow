package pivotaltracker

import (
	// Internal
	"github.com/salsaflow/salsaflow/config"
	"github.com/salsaflow/salsaflow/errs"
)

const Id = "pivotal_tracker"

const LocalConfigTemplate = `
#pivotal_tracker:
#  project_id: 123456
#
#  All config that follows is OPTIONAL.
#
#  # You can set the following key to filter the stories
#  # SalsaFlow will be operating with when inside of this repository.
#  # This is handy in case you are using a single PT projects
#  # with multiple repositories.
#  component_label: "server"
#
#  # The values visible there are the default choices.
#  # In case the defaults are fine, just delete the section,
#  # otherwise uncomment what you need to change.
#  workflow_labels:
#    point_me: "point_me"
#    reviewed: "reviewed"
#    skip_review: "no review"
#    tested: "qa+"
#    skip_testing: "no qa"
#    skip_check_labels:
#      - "wontfix"
#      - "dupe"
`

const (
	DefaultPointMeLabel     = "point me"
	DefaultReviewedLabel    = "reviewed"
	DefaultSkipReviewLabel  = "no review"
	DefaultTestedLabel      = "qa+"
	DefaultSkipTestingLabel = "no qa"
)

var DefaultSkipCheckLabels = []string{"dupe", "wontfix"}

// Local configuration ---------------------------------------------------------

type LocalConfig struct {
	PT struct {
		ProjectId      int    `yaml:"project_id"`
		ComponentLabel string `yaml:"component_label"`
		Labels         struct {
			PointMeLabel     string   `yaml:"point_me"`
			ReviewedLabel    string   `yaml:"reviewed"`
			SkipReviewLabel  string   `yaml:"skip_review"`
			TestedLabel      string   `yaml:"tested"`
			SkipTestingLabel string   `yaml:"skip_testing"`
			SkipCheckLabels  []string `yaml:"skip_check_labels"`
		} `yaml:"workflow_labels"`
	} `yaml:"pivotal_tracker"`
}

func (local *LocalConfig) validate() error {
	var (
		task = "Validate the local Pivotal Tracker configuration"
		pt   = &local.PT
		err  error
	)
	switch {
	case pt.ProjectId == 0:
		err = &config.ErrKeyNotSet{Id + ".project_id"}
	case pt.Labels.PointMeLabel == "":
		err = &config.ErrKeyNotSet{Id + ".labels.point_me"}
	case pt.Labels.ReviewedLabel == "":
		err = &config.ErrKeyNotSet{Id + ".labels.reviewed"}
	case pt.Labels.SkipReviewLabel == "":
		err = &config.ErrKeyNotSet{Id + ".labels.skip_review"}
	case pt.Labels.TestedLabel == "":
		err = &config.ErrKeyNotSet{Id + ".labels.tested"}
	case pt.Labels.SkipTestingLabel == "":
		err = &config.ErrKeyNotSet{Id + ".labels.skip_testing"}
	}
	if err != nil {
		return errs.NewError(task, err)
	}
	return nil
}

// Global configuration --------------------------------------------------------

type GlobalConfig struct {
	PT struct {
		UserToken string `yaml:"token"`
	} `yaml:"pivotal_tracker"`
}

func (global *GlobalConfig) validate() error {
	var (
		task = "Validate the global Pivotal Tracker configuration"
		pt   = &global.PT
		err  error
	)
	switch {
	case pt.UserToken == "":
		err = &config.ErrKeyNotSet{Id + ".token"}
	}
	if err != nil {
		return errs.NewError(task, err)
	}
	return nil
}

// Proxy object ----------------------------------------------------------------

type Config interface {
	ProjectId() int
	ComponentLabel() string
	PointMeLabel() string
	ReviewedLabel() string
	SkipReviewLabel() string
	TestedLabel() string
	SkipTestingLabel() string
	SkipCheckLabels() []string
	UserToken() string
}

var configCache Config

func LoadConfig() (Config, error) {
	// Try the cache first.
	if configCache != nil {
		return configCache, nil
	}

	// Load the local config.
	var local LocalConfig
	if err := config.UnmarshalLocalConfig(&local); err != nil {
		return nil, err
	}

	labels := &local.PT.Labels
	if labels.PointMeLabel == "" {
		labels.PointMeLabel = DefaultPointMeLabel
	}
	if labels.ReviewedLabel == "" {
		labels.ReviewedLabel = DefaultReviewedLabel
	}
	if labels.SkipReviewLabel == "" {
		labels.SkipReviewLabel = DefaultSkipReviewLabel
	}
	if labels.TestedLabel == "" {
		labels.TestedLabel = DefaultTestedLabel
	}
	if labels.SkipTestingLabel == "" {
		labels.SkipTestingLabel = DefaultSkipTestingLabel
	}
	labels.SkipCheckLabels = append(labels.SkipCheckLabels, DefaultSkipCheckLabels...)

	if err := local.validate(); err != nil {
		return nil, err
	}

	// Load the global config.
	var global GlobalConfig
	if err := config.UnmarshalGlobalConfig(&global); err != nil {
		return nil, err
	}
	if err := global.validate(); err != nil {
		return nil, err
	}

	configCache = &configProxy{&local, &global}
	return configCache, nil
}

type configProxy struct {
	local  *LocalConfig
	global *GlobalConfig
}

func (proxy *configProxy) ProjectId() int {
	return proxy.local.PT.ProjectId
}

func (proxy *configProxy) ComponentLabel() string {
	return proxy.local.PT.ComponentLabel
}

func (proxy *configProxy) PointMeLabel() string {
	return proxy.local.PT.Labels.PointMeLabel
}

func (proxy *configProxy) ReviewedLabel() string {
	return proxy.local.PT.Labels.ReviewedLabel
}

func (proxy *configProxy) SkipReviewLabel() string {
	return proxy.local.PT.Labels.SkipReviewLabel
}

func (proxy *configProxy) TestedLabel() string {
	return proxy.local.PT.Labels.TestedLabel
}

func (proxy *configProxy) SkipTestingLabel() string {
	return proxy.local.PT.Labels.SkipTestingLabel
}

func (proxy *configProxy) SkipCheckLabels() []string {
	return proxy.local.PT.Labels.SkipCheckLabels
}

func (proxy *configProxy) UserToken() string {
	return proxy.global.PT.UserToken
}
