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
	task := "Check whether the repository has been initialised"
	versionString, err := git.GetConfigString("salsaflow.initialised")
	if err != nil {
		return errs.NewError(task, err, nil)
	}
	if versionString == metadata.Version {
		return errs.NewError(task, ErrInitialised, nil)
	}

	log.Log("Initialising the repository for SalsaFlow")

	// Make sure the user is using the right version of Git.
	//
	// The check is here and not in app.Init because it is highly improbable
	// that the check would pass once and then fail later. Once the right
	// version of git is installed, it most probably stays.
	task = "Check the git version being used"
	log.Run(task)
	stdout, err := git.Run("--version")
	if err != nil {
		return errs.NewError(task, err, nil)
	}
	pattern := regexp.MustCompile("^git version (([0-9]+)[.]([0-9]+).*)")
	parts := pattern.FindStringSubmatch(stdout.String())
	if len(parts) != 4 {
		return errs.NewError(task, errors.New("unexpected git --version output"), nil)
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
			task,
			errors.New("unsupported git version detected: "+gitVersion),
			bytes.NewBufferString(hint))
	}

	// Get hold of a git config instance.
	gitConfig, err := git.LoadConfig()
	if err != nil {
		return err
	}

	// Make sure that the master branch exists.
	task = "Make sure the master branch exists"
	log.Run(task)
	var stableBranch = gitConfig.StableBranchName()
	exists, err := git.RefExists(stableBranch)
	if err != nil {
		return errs.NewError(task, err, nil)
	}
	if !exists {
		stderr := bytes.NewBufferString(fmt.Sprintf(
			"Make sure that branch '%v' exists and run init again.\n", stableBranch))
		err := fmt.Errorf("branch '%v' not found", stableBranch)
		return errs.NewError(task, err, stderr)
	}

	// Make sure that the trunk branch exists.
	task = "Make sure the trunk branch exists"
	log.Run(task)
	var trunkBranch = gitConfig.TrunkBranchName()
	exists, err = git.RefExists(trunkBranch)
	if err != nil {
		return errs.NewError(task, err, nil)
	}
	if !exists {
		task := "Create the trunk branch"
		log.Log(fmt.Sprintf(
			"No branch '%s' found. Will create one for you for free!", trunkBranch))
		log.NewLine(fmt.Sprintf(
			"The newly created branch is pointing to '%v'.", stableBranch))
		if err := git.Branch(trunkBranch, stableBranch); err != nil {
			return errs.NewError(task, err, nil)
		}

		task = "Push the newly created trunk branch"
		log.Run(task)
		if err := git.Push(gitConfig.RemoteName(), trunkBranch+":"+trunkBranch); err != nil {
			return errs.NewError(task, err, nil)
		}
	}

	// Verify our git hooks are installed and used.
	task = "Check the current git commit-msg hook"
	log.Run(task)
	if err := hooks.CheckAndUpsert(hooks.HookTypeCommitMsg); err != nil {
		return errs.NewError(task, err, nil)
	}

	task = "Check the current git pre-push hook"
	log.Run(task)
	if err := hooks.CheckAndUpsert(hooks.HookTypePrePush); err != nil {
		return errs.NewError(task, err, nil)
	}

	// Run other registered init hooks.
	task = "Running the registered repository init hooks"
	log.Log(task)
	if err := executeInitHooks(); err != nil {
		return errs.NewError(task, err, nil)
	}

	// Success! Mark the repository as initialised in git config.
	task = "Mark the repository as initialised"
	if err := git.SetConfigString("salsaflow.initialised", metadata.Version); err != nil {
		return err
	}
	asciiart.PrintThumbsUp()
	fmt.Println()
	log.Log("The repository is initialised")

	return nil
}
