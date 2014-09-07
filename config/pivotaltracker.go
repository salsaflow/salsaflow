package config

import (
	"github.com/tchap/git-trunk/log"
)

var PivotalTracker PivotalTrackerConfig

var ptLocalConfig struct {
	PT struct {
		ProjectId int `yaml:"ProjectId"`
	} `yaml:"PivotalTracker"`
}

var ptGlobalConfig struct {
	PT struct {
		Token string `yaml:"Token"`
	} `yaml:"PivotalTracker"`
}

func init() {
	if err := fillLocalConfig(&ptLocalConfig); err != nil {
		log.Fail("Load local Pivotal Tracker configuration")
		log.Fatalln(err)
	}

	if err := fillGlobalConfig(&ptGlobalConfig); err != nil {
		log.Fail("Load global Pivotal Tracker configuration")
		log.Fatalln(err)
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
	return ptLocalConfig.PT.ProjectId
}

func (pt *PivotalTrackerConfig) ApiToken() string {
	return ptGlobalConfig.PT.Token
}

func ptValidateLocalConfig() error {
	switch {
	case ptLocalConfig.PT.ProjectId == 0:
		return &ErrFieldNotSet{"PivotalTracker.ProjectId"}
	default:
		return nil
	}
}

func ptValidateGlobalConfig() error {
	switch {
	case ptGlobalConfig.PT.Token == "":
		return &ErrFieldNotSet{"PivotalTracker.Token"}
	default:
		return nil
	}
}
