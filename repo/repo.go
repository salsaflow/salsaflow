package repo

import (
	// Stdlib
	"errors"
	"fmt"
	"regexp"
	"strconv"

	// Internal
	"github.com/salsaflow/salsaflow/app/metadata"
	"github.com/salsaflow/salsaflow/asciiart"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/git"
	"github.com/salsaflow/salsaflow/hooks"
	"github.com/salsaflow/salsaflow/log"
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

func Init(force bool) error {
	// Check whether the repository has been initialised yet.
	task := "Check whether the repository has been initialised"
	versionString, err := git.GetConfigString("salsaflow.initialised")
	if err != nil {
		return errs.NewError(task, err)
	}
	if versionString == metadata.Version && !force {
		return errs.NewError(task, ErrInitialised)
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
		return errs.NewError(task, err)
	}
	pattern := regexp.MustCompile("^git version (([0-9]+)[.]([0-9]+).*)")
	parts := pattern.FindStringSubmatch(stdout.String())
	if len(parts) != 4 {
		return errs.NewError(task, errors.New("unexpected git --version output"))
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
		return errs.NewErrorWithHint(
			task,
			errors.New("unsupported git version detected: "+gitVersion),
			hint)
	}

	// Get hold of a git config instance.
	gitConfig, err := git.LoadConfig()
	if err != nil {
		return err
	}
	var (
		remoteName   = gitConfig.RemoteName()
		trunkBranch  = gitConfig.TrunkBranchName()
		stableBranch = gitConfig.StableBranchName()
	)

	// Make sure that the master branch exists.
	task = fmt.Sprintf("Make sure branch '%v' exists", stableBranch)
	log.Run(task)
	err = git.EnsureLocalTrackingBranch(stableBranch, remoteName)
	if err != nil {
		if ex, ok := err.(*git.ErrRefNotFound); ok {
			hint := fmt.Sprintf(
				"Make sure that branch '%v' exists and run init again.\n", ex.Ref())
			return errs.NewErrorWithHint(task, err, hint)
		}
		return errs.NewError(task, err)
	}

	// Make sure that the trunk branch exists.
	task = fmt.Sprintf("Make sure branch '%v' exists", trunkBranch)
	log.Run(task)
	err = git.EnsureLocalTrackingBranch(trunkBranch, remoteName)
	if err != nil {
		if _, ok := err.(*git.ErrRefNotFound); ok {
			task := fmt.Sprintf("Create branch '%v'", trunkBranch)
			log.Log(fmt.Sprintf(
				"Branch '%v' not found. Will create one for you for free!", trunkBranch))
			if err := git.Branch(trunkBranch, stableBranch); err != nil {
				return errs.NewError(task, err)
			}
			log.NewLine(fmt.Sprintf(
				"The newly created branch is pointing to '%v'.", stableBranch))

			task = fmt.Sprintf("Push branch '%v' to remote '%v'", trunkBranch, remoteName)
			log.Run(task)
			if err := git.Push(remoteName, trunkBranch+":"+trunkBranch); err != nil {
				return errs.NewError(task, err)
			}
		}
		return errs.NewError(task, err)
	}

	// Verify our git hooks are installed and used.
	for _, kind := range hooks.HookTypes {
		task := fmt.Sprintf("Check the current git %v hook", kind)
		log.Run(task)
		if err := hooks.CheckAndUpsert(kind, force); err != nil {
			return errs.NewError(task, err)
		}
	}

	// Run other registered init hooks.
	task = "Running the registered repository init hooks"
	log.Log(task)
	if err := executeInitHooks(); err != nil {
		return errs.NewError(task, err)
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
