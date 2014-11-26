package pivotaltracker

import (
	// Stdlib
	"errors"

	// Internal
	cfg "github.com/salsaflow/salsaflow/config"
	"github.com/salsaflow/salsaflow/log"
)

const (
	Id = "pivotal_tracker"

	DefaultPointMeLabel  = "point me"
	DefaultReviewedLabel = "reviewed"
	DefaultVerifiedLabel = "qa+"
)

var DefaultSkipLabels = []string{"dupe", "wontfix"}

func loadConfig() error {
	if err := loadGlobalConfig(); err != nil {
		return err
	}
	return loadLocalConfig()
}

// Global configuration --------------------------------------------------------

type ptGlobalConfig struct {
	Token string `yaml:"token"`
}

func (config *ptGlobalConfig) Validate() error {
	switch {
	case config.Token == "":
		return &cfg.ErrKeyNotSet{Id + ".token"}
	default:
		return nil
	}
}

var ptGlobalWrapper struct {
	C *ptGlobalConfig `yaml:"pivotal_tracker"`
}

func loadGlobalConfig() error {
	task := "Load global Pivotal Tracker configuration"
	if err := cfg.FillGlobalConfig(&ptGlobalWrapper); err != nil {
		log.Fail(task)
		return err
	}

	if ptGlobalWrapper.C == nil {
		log.Fail(task)
		return errors.New("Pivotal Tracker global configuration section missing")
	}

	if err := ptGlobalWrapper.C.Validate(); err != nil {
		log.Fail(task)
		return err
	}

	return nil
}

// Local configuration ---------------------------------------------------------

type ptLocalConfig struct {
	ProjectId int `yaml:"project_id"`
	Labels    struct {
		PointMeLabel    string   `yaml:"point_me"`
		ReviewedLabel   string   `yaml:"reviewed"`
		VerifiedLabel   string   `yaml:"verified"`
		SkipCheckLabels []string `yaml:"skip_release_check"`
	} `yaml:"labels"`
}

func (config *ptLocalConfig) Validate() error {
	switch {
	case config.ProjectId == 0:
		return &cfg.ErrKeyNotSet{Id + ".project_id"}
	case config.Labels.PointMeLabel == "":
		return &cfg.ErrKeyNotSet{Id + ".labels.point_me"}
	case config.Labels.ReviewedLabel == "":
		return &cfg.ErrKeyNotSet{Id + ".labels.reviewed"}
	case config.Labels.VerifiedLabel == "":
		return &cfg.ErrKeyNotSet{Id + ".labels.verified"}
	default:
		return nil
	}
}

var ptLocalWrapper struct {
	C *ptLocalConfig `yaml:"pivotal_tracker"`
}

func loadLocalConfig() error {
	task := "Load local Pivotal Tracker configuration"
	if err := cfg.FillLocalConfig(&ptLocalWrapper); err != nil {
		log.Fail(task)
		return err
	}

	config := ptLocalWrapper.C
	if config == nil {
		log.Fail(task)
		return errors.New("Pivotal Tracker local configuration section missing")
	}

	if config.Labels.PointMeLabel == "" {
		config.Labels.PointMeLabel = DefaultPointMeLabel
	}
	if config.Labels.ReviewedLabel == "" {
		config.Labels.ReviewedLabel = DefaultReviewedLabel
	}
	if config.Labels.VerifiedLabel == "" {
		config.Labels.VerifiedLabel = DefaultVerifiedLabel
	}
	config.Labels.SkipCheckLabels = append(config.Labels.SkipCheckLabels, DefaultSkipLabels...)

	if err := config.Validate(); err != nil {
		log.Fail(task)
		return err
	}

	return nil
}

// Config proxy object ---------------------------------------------------------

var config ptConfig

type ptConfig struct{}

func (pt *ptConfig) ProjectId() int {
	return ptLocalWrapper.C.ProjectId
}

func (pt *ptConfig) PointMeLabel() string {
	return ptLocalWrapper.C.Labels.PointMeLabel
}

func (pt *ptConfig) ReviewedLabel() string {
	return ptLocalWrapper.C.Labels.ReviewedLabel
}

func (pt *ptConfig) VerifiedLabel() string {
	return ptLocalWrapper.C.Labels.VerifiedLabel
}

func (pt *ptConfig) SkipCheckLabels() []string {
	return ptLocalWrapper.C.Labels.SkipCheckLabels
}

func (pt *ptConfig) ApiToken() string {
	return ptGlobalWrapper.C.Token
}
