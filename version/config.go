package version

import (
	// Stdlib
	"fmt"

	// Internal
	"github.com/salsaflow/salsaflow/config/loader"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/prompt"

	// Vendor
	"github.com/blang/semver"
	"github.com/fatih/color"
)

func init() {
	loader.RegisterBootstrapConfigSpec(newConfigSpec())
}

// Configuration ===============================================================

const ConfigKey = "salsaflow.core.versioning"

type Config struct {
	TrunkSuffix   semver.PRVersion
	TestingSuffix semver.PRVersion
	StagingSuffix semver.PRVersion
}

// LoadConfig can be used to load versioning-related configuration for SalsaFlow.
func LoadConfig() (*Config, error) {
	task := "Load versioning-related configuration"
	spec := newConfigSpec()
	if err := loader.LoadConfig(spec); err != nil {
		return nil, errs.NewError(task, err)
	}
	return spec.local.parse()
}

// Configuration spec ----------------------------------------------------------

func newConfigSpec() *configSpec {
	return &configSpec{}
}

type configSpec struct {
	local *LocalConfig
}

func (spec *configSpec) ConfigKey() string {
	return ConfigKey
}

func (spec *configSpec) GlobalConfig() loader.ConfigContainer {
	return nil
}

func (spec *configSpec) LocalConfig() loader.ConfigContainer {
	spec.local = &LocalConfig{}
	return spec.local
}

// Local config ----------------------------------------------------------------

type LocalConfig struct {
	TrunkSuffix   string `prompt:"trunk version string suffix"   default:"dev"   json:"trunk_suffix"`
	TestingSuffix string `prompt:"testing version string suffix" default:"qa"    json:"testing_suffix"`
	StagingSuffix string `prompt:"staging version string suffix" default:"stage" json:"staging_suffix"`
}

func (local *LocalConfig) PromptUserForConfig() error {
	task := "Prompt the user for local versioning-related configuration"

	for {
		var c LocalConfig
		if err := prompt.Dialog(&c, "Insert the"); err != nil {
			return errs.NewError(task, err)
		}

		if _, err := c.parse(); err != nil {
			fmt.Println()
			color.Yellow("Invalid version suffix inserted, please try again!")
			fmt.Println()
			continue
		}

		*local = c
		return nil
	}
}

func (local *LocalConfig) Validate() error {
	_, err := local.parse()
	return err
}

func (local *LocalConfig) parse() (*Config, error) {
	task := "Parse the version string suffixes"

	var (
		config Config
		err    error
	)
	parse := func(dst *semver.PRVersion, src string) {
		if err != nil {
			return
		}
		var v semver.PRVersion
		v, err = semver.NewPRVersion(src)
		if err == nil {
			*dst = v
		}
	}
	parse(&config.TrunkSuffix, local.TrunkSuffix)
	parse(&config.TestingSuffix, local.TestingSuffix)
	parse(&config.StagingSuffix, local.StagingSuffix)
	if err != nil {
		return nil, errs.NewError(task, err)
	}

	return &config, nil
}
