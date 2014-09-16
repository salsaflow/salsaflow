package config

import (
	"errors"
)

const (
	sectionPivotalTracker = "pivotal_tracker"

	ptDefaultPointMeLabel  = "point me"
	ptDefaultReviewedLabel = "reviewed"
	ptDefaultVerifiedLabel = "qa+"
)

var ptDefaultSkipLabels = []string{"dupe", "wontfix"}

func mustInitPivotalTracker() {
	mustInitPivotalTrackerGlobal()
	mustInitPivotalTrackerLocal()
}

// Global configuration --------------------------------------------------------

type ptGlobalConfig struct {
	Token string `yaml:"token"`
}

func (config *ptGlobalConfig) Validate() error {
	switch {
	case config.Token == "":
		return &ErrKeyNotSet{sectionPivotalTracker + ".token"}
	default:
		return nil
	}
}

var ptGlobalWrapper struct {
	C *ptGlobalConfig `yaml:"pivotal_tracker"`
}

func mustInitPivotalTrackerGlobal() {
	msg := "Parse global Pivotal Tracker configuration"
	if err := fillGlobalConfig(&ptGlobalWrapper); err != nil {
		die(msg, err)
	}

	if ptGlobalWrapper.C == nil {
		die(msg, errors.New("Pivotal Tracker global configuration section missing"))
	}

	if err := ptGlobalWrapper.C.Validate(); err != nil {
		die(msg, err)
	}
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
		return &ErrKeyNotSet{sectionPivotalTracker + ".project_id"}
	case config.Labels.PointMeLabel == "":
		return &ErrKeyNotSet{sectionPivotalTracker + ".labels.point_me"}
	case config.Labels.ReviewedLabel == "":
		return &ErrKeyNotSet{sectionPivotalTracker + ".labels.reviewed"}
	case config.Labels.VerifiedLabel == "":
		return &ErrKeyNotSet{sectionPivotalTracker + ".labels.verified"}
	default:
		return nil
	}
}

var ptLocalWrapper struct {
	C *ptLocalConfig `yaml:"pivotal_tracker"`
}

func mustInitPivotalTrackerLocal() {
	msg := "Parse local Pivotal Tracker configuration"
	if err := fillLocalConfig(&ptLocalWrapper); err != nil {
		die(msg, err)
	}

	config := ptLocalWrapper.C
	if config == nil {
		die(msg, errors.New("Pivotal Tracker local configuration section missing"))
	}

	if config.Labels.PointMeLabel == "" {
		config.Labels.PointMeLabel = ptDefaultPointMeLabel
	}
	if config.Labels.ReviewedLabel == "" {
		config.Labels.ReviewedLabel = ptDefaultReviewedLabel
	}
	if config.Labels.VerifiedLabel == "" {
		config.Labels.VerifiedLabel = ptDefaultVerifiedLabel
	}
	config.Labels.SkipCheckLabels = append(config.Labels.SkipCheckLabels, ptDefaultSkipLabels...)

	if err := config.Validate(); err != nil {
		die(msg, err)
	}
}

// Config proxy object ---------------------------------------------------------

var PivotalTracker PivotalTrackerConfig

type PivotalTrackerConfig struct{}

func (pt *PivotalTrackerConfig) ProjectId() int {
	return ptLocalWrapper.C.ProjectId
}

func (pt *PivotalTrackerConfig) PointMeLabel() string {
	return ptLocalWrapper.C.Labels.PointMeLabel
}

func (pt *PivotalTrackerConfig) ReviewedLabel() string {
	return ptLocalWrapper.C.Labels.ReviewedLabel
}

func (pt *PivotalTrackerConfig) VerifiedLabel() string {
	return ptLocalWrapper.C.Labels.VerifiedLabel
}

func (pt *PivotalTrackerConfig) SkipCheckLabels() []string {
	return ptLocalWrapper.C.Labels.SkipCheckLabels
}

func (pt *PivotalTrackerConfig) ApiToken() string {
	return ptGlobalWrapper.C.Token
}
