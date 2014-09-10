package config

import (
	"github.com/tchap/git-trunk/log"
)

const (
	DefaultReviewedLabel = "reviewed"
	DefaultVerifiedLabel = "qa+"
)

var PivotalTracker PivotalTrackerConfig

var ptLocalConfig struct {
	PT struct {
		ProjectId int `yaml:"project_id"`
		Labels    struct {
			ReviewedLabel string `yaml:"reviewed"`
			VerifiedLabel string `yaml:"verified"`
		} `yaml:"labels"`
	} `yaml:"pivotal_tracker"`
}

var ptLocal = &ptLocalConfig.PT

var ptGlobalConfig struct {
	PT struct {
		Token string `yaml:"token"`
	} `yaml:"pivotal_tracker"`
}

var ptGlobal = &ptGlobalConfig.PT

func init() {
	if err := fillLocalConfig(&ptLocalConfig); err != nil {
		log.Fail("Load local Pivotal Tracker configuration")
		log.Fatalln(err)
	}

	if err := fillGlobalConfig(&ptGlobalConfig); err != nil {
		log.Fail("Load global Pivotal Tracker configuration")
		log.Fatalln(err)
	}

	if ptLocal.Labels.ReviewedLabel == "" {
		ptLocal.Labels.ReviewedLabel = DefaultReviewedLabel
	}
	if ptLocal.Labels.VerifiedLabel == "" {
		ptLocal.Labels.VerifiedLabel = DefaultVerifiedLabel
	}

	if err := ptValidateLocalConfig(); err != nil {
		log.Fail("Validate local Pivotal Tracker configuration")
		log.Fatalln(err)
	}

	if err := ptValidateGlobalConfig(); err != nil {
		log.Fail("Validate global Pivotal Tracker configuration")
		log.Fatalln(err)
	}
}

type PivotalTrackerConfig struct{}

func (pt *PivotalTrackerConfig) ProjectId() int {
	return ptLocal.ProjectId
}

func (pt *PivotalTrackerConfig) ReviewedLabel() string {
	return ptLocal.Labels.ReviewedLabel
}

func (pt *PivotalTrackerConfig) VerifiedLabel() string {
	return ptLocal.Labels.VerifiedLabel
}

func (pt *PivotalTrackerConfig) ApiToken() string {
	return ptGlobal.Token
}

func ptValidateLocalConfig() error {
	switch {
	case ptLocal.ProjectId == 0:
		return &ErrFieldNotSet{"PivotalTracker.ProjectId"}
	default:
		return nil
	}
}

func ptValidateGlobalConfig() error {
	switch {
	case ptGlobal.Token == "":
		return &ErrFieldNotSet{"PivotalTracker.Token"}
	default:
		return nil
	}
}
