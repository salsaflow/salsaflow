package repo

import (
	// Stdlib
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strconv"

	// Internal
	"github.com/salsita/salsaflow/app/metadata"
	"github.com/salsita/salsaflow/asciiart"
	"github.com/salsita/salsaflow/config"
	"github.com/salsita/salsaflow/errs"
	"github.com/salsita/salsaflow/git"
	"github.com/salsita/salsaflow/hooks"
	"github.com/salsita/salsaflow/log"
)

var initHooks []InitHook

type InitHook func() error

func AddInitHook(hook InitHook) {
	initHooks = append(initHooks, hook)
}

func executeInitHooks() error {
	for _, hook := range initHooks {
		if err := hook(); err != nil {
			return err
		}
	}
	return nil
}

var ErrInitialised = errors.New("repository already initialised")

func Init() error {
	// Check whether the repository has been initialised yet.
	msg := "Check whether the repository has been initialised"
	versionString, err := git.GetConfigString("salsaflow.initialised")
	if err != nil {
		return errs.NewError(msg, err, nil)
	}
	if versionString == metadata.Version {
		return errs.NewError(msg, ErrInitialised, nil)
	}

	log.Log("Initialising the repository for SalsaFlow")

	// Make sure the user is using the right version of Git.
	//
	// The check is here and not in app.Init because it is highly improbable
	// that the check would pass once and then fail later. Once the right
	// version of git is installed, it most probably stays.
	msg = "Check the git version being used"
	log.Run(msg)
	stdout, stderr, err := git.Run("--version")
	if err != nil {
		return errs.NewError(msg, err, stderr)
	}
	pattern := regexp.MustCompile("^git version (([0-9]+)[.]([0-9]+).*)")
	parts := pattern.FindStringSubmatch(stdout.String())
	if len(parts) != 4 {
		return errs.NewError(msg, errors.New("unexpected git --version output"), nil)
	}
	gitVersion := parts[1]
	// This cannot fail since we matched the regexp.
	major, _ := strconv.Atoi(parts[2])
	minor, _ := strconv.Atoi(parts[3])
	// We need Git version 1.8.5.4+, so let's require 1.9+.
	switch {
	case major >= 2:
		// OK
	case major == 1 && minor >= 9:
		// OK
	default:
		hint := `
You need Git version 1.9.0 or newer.

`
		return errs.NewError(
			msg,
			errors.New("unsupported git version detected: "+gitVersion),
			bytes.NewBufferString(hint))
	}

	// Make sure that the master branch exists.
	msg = "Make sure the master branch exists"
	log.Run(msg)
	exists, stderr, err := git.RefExists(config.MasterBranch)
	if err != nil {
		return errs.NewError(msg, err, stderr)
	}
	if !exists {
		stderr := bytes.NewBufferString(fmt.Sprintf(
			"Make sure that branch '%v' exists and run init again.", config.MasterBranch))
		err := fmt.Errorf("branch '%v' not found", config.MasterBranch)
		return errs.NewError(msg, err, stderr)
	}

	// Make sure that the trunk branch exists.
	msg = "Make sure the trunk branch exists"
	log.Run(msg)
	exists, stderr, err = git.RefExists(config.TrunkBranch)
	if err != nil {
		return errs.NewError(msg, err, stderr)
	}
	if !exists {
		msg := "Create the trunk branch"
		log.Log(fmt.Sprintf(
			"No branch '%s' found. Will create one for you for free!", config.TrunkBranch))
		log.NewLine(fmt.Sprintf(
			"The newly created branch is pointing to '%v'.", config.MasterBranch))
		stderr, err := git.Branch(config.TrunkBranch, config.MasterBranch)
		if err != nil {
			return errs.NewError(msg, err, stderr)
		}

		msg = "Push the newly created trunk branch"
		log.Run(msg)
		stderr, err = git.Push(config.OriginName, config.TrunkBranch+":"+config.TrunkBranch)
		if err != nil {
			return errs.NewError(msg, err, stderr)
		}
	}

	// Check the global configuration file.
	msg = "Check the global SalsaFlow configuration"
	log.Run(msg)
	if _, err := config.ReadGlobalConfig(); err != nil {
		return errs.NewError(
			msg,
			fmt.Errorf("could not read config file '%v': %v",
				"$HOME/"+config.GlobalConfigFileName, err),
			nil)
	}

	// Check the project-specific configuration file.
	msg = "Check the local SalsaFlow configuration"
	log.Run(msg)
	if _, stderr, err = config.ReadLocalConfig(); err != nil {
		return errs.NewError(msg,
			fmt.Errorf("could not read config file '%v' on branch '%v': %v",
				config.LocalConfigFileName, config.ConfigBranch, err),
			stderr)
	}

	// Verify our git hooks are installed and used.
	msg = "Check the current git commit-msg hook"
	log.Run(msg)
	if err := hooks.CheckAndUpsert(hooks.HookTypeCommitMsg); err != nil {
		return errs.NewError(msg, err, nil)
	}

	msg = "Check the current git pre-push hook"
	log.Run(msg)
	if err := hooks.CheckAndUpsert(hooks.HookTypePrePush); err != nil {
		return errs.NewError(msg, err, nil)
	}

	// Run other registered init hooks.
	msg = "Running the registered repository init hooks"
	log.Log(msg)
	if err := executeInitHooks(); err != nil {
		return errs.NewError(msg, err, nil)
	}

	// Success! Mark the repository as initialised in git config.
	msg = "Mark the repository as initialised"
	if err := git.SetConfigString("salsaflow.initialised", metadata.Version); err != nil {
		return err
	}
	asciiart.PrintThumbsUp()
	fmt.Println()
	log.Log("The repository is initialised")

	return nil
}
