package pivotaltracker

import (
	// Stdlib
	"regexp"

	// Internal
	"github.com/salsaflow/salsaflow/config"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/git"
)

const Id = "pivotal_tracker"

const LocalConfigTemplate = `
#pivotal_tracker:
#  project_id: 123456
#
#  THE FOLLOWING SECTION IS OPTIONAL.
#  The values visible there are the default choices.
#  In case the defaults are fine, just delete the section,
#  otherwise uncomment what you need to change.
#
#  labels:
#    point_me: "point me"
#    no_review: "no review"
#    reviewed: "reviewed"
#    verified: "qa+"
#    skip_check_labels:
#      - "wontfix"
#      - "dupe"
`

const (
	DefaultPointMeLabel     = "point me"
	DefaultImplementedLabel = "implemented"
	DefaultNoReviewLabel    = "no review"
	DefaultReviewedLabel    = "reviewed"
)

var DefaultSkipCheckLabels = []string{"dupe", "wontfix"}

// Local configuration ---------------------------------------------------------

type LocalConfig struct {
	PT struct {
		ProjectId int `yaml:"project_id"`
		Labels    struct {
			PointMeLabel     string   `yaml:"point_me"`
			ImplementedLabel string   `yaml:"implemented"`
			NoReviewLabel    string   `yaml:"no_review"`
			ReviewedLabel    string   `yaml:"reviewed"`
			SkipCheckLabels  []string `yaml:"skip_check_labels"`
		} `yaml:"labels"`
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
	case pt.Labels.ImplementedLabel == "":
		err = &config.ErrKeyNotSet{Id + ".labels.implemented"}
	case pt.Labels.NoReviewLabel == "":
		err = &config.ErrKeyNotSet{Id + ".labels.no_review"}
	case pt.Labels.ReviewedLabel == "":
		err = &config.ErrKeyNotSet{Id + ".labels.reviewed"}
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
	PointMeLabel() string
	ImplementedLabel() string
	NoReviewLabel() string
	ReviewedLabel() string
	SkipCheckLabels() []string
	UserToken() string
	IncludeStoryLabelFilter() *regexp.Regexp
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
	if labels.ImplementedLabel == "" {
		labels.ImplementedLabel = DefaultImplementedLabel
	}
	if labels.NoReviewLabel == "" {
		labels.NoReviewLabel = DefaultNoReviewLabel
	}
	if labels.ReviewedLabel == "" {
		labels.ReviewedLabel = DefaultReviewedLabel
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

	// Load git config.
	storyFilter, err := git.GetConfigString("salsaflow.pivotaltracker.include-stories")
	if err != nil {
		return nil, err
	}

	var storyFilterRe *regexp.Regexp
	if storyFilter != "" {
		var err error
		storyFilterRe, err = regexp.Compile(storyFilter)
		if err != nil {
			return nil, err
		}
	}

	configCache = &configProxy{&local, &global, storyFilterRe}
	return configCache, nil
}

type configProxy struct {
	local  *LocalConfig
	global *GlobalConfig

	storyFilter *regexp.Regexp
}

func (proxy *configProxy) ProjectId() int {
	return proxy.local.PT.ProjectId
}

func (proxy *configProxy) PointMeLabel() string {
	return proxy.local.PT.Labels.PointMeLabel
}

func (proxy *configProxy) ImplementedLabel() string {
	return proxy.local.PT.Labels.ImplementedLabel
}

func (proxy *configProxy) NoReviewLabel() string {
	return proxy.local.PT.Labels.NoReviewLabel
}

func (proxy *configProxy) ReviewedLabel() string {
	return proxy.local.PT.Labels.ReviewedLabel
}

func (proxy *configProxy) SkipCheckLabels() []string {
	return proxy.local.PT.Labels.SkipCheckLabels
}

func (proxy *configProxy) UserToken() string {
	return proxy.global.PT.UserToken
}

func (proxy *configProxy) IncludeStoryLabelFilter() *regexp.Regexp {
	return proxy.storyFilter
}
