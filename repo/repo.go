package repo

import (
	// Stdlib
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	// Internal
	"github.com/salsita/salsaflow/asciiart"
	"github.com/salsita/salsaflow/config"
	"github.com/salsita/salsaflow/errs"
	"github.com/salsita/salsaflow/git"
	"github.com/salsita/salsaflow/log"
	"github.com/salsita/salsaflow/prompt"
	"github.com/salsita/salsaflow/shell"

	// Other
	"bitbucket.org/kardianos/osext"
)

var CommitMsgHookFileName = "salsaflow-commit-msg"

func getCommitMsgHookFileName() string {
	if runtime.GOOS == "windows" && !strings.HasSuffix(CommitMsgHookFileName, ".exe") {
		return CommitMsgHookFileName + ".exe"
	}
	return CommitMsgHookFileName
}

func Init() *errs.Error {
	// Check whether the repository has been initialized yet.
	msg := "Check whether the repository has been initialized"
	initialized, stderr, err := git.GetConfigBool("salsaflow.initialized")
	if err != nil {
		return errs.NewError(msg, stderr, err)
	}
	if initialized {
		// There is nothing to do.
		return nil
	}

	log.Log("Initialising the repository for SalsaFlow")

	// Make sure the user is using the right version of Git.
	//
	// The check is here and not in app.Init because it is highly improbable
	// that the check would pass once and then fail later. Once the right
	// version of git is installed, it most probably stays.
	msg = "Check the git version being used"
	log.Run(msg)
	stdout, stderr, err := git.Git("--version")
	if err != nil {
		return errs.NewError(msg, stderr, err)
	}
	pattern := regexp.MustCompile("^git version (([0-9]+)[.]([0-9]+).*)")
	parts := pattern.FindStringSubmatch(stdout.String())
	if len(parts) != 4 {
		return errs.NewError(msg, nil, errors.New("unexpected git --version output"))
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
			bytes.NewBufferString(hint),
			errors.New("unsupported git version detected: "+gitVersion))
	}

	// Make sure that the master branch exists.
	msg = "Make sure the master branch exists"
	log.Run(msg)
	exists, stderr, err := git.RefExists(config.MasterBranch)
	if err != nil {
		return errs.NewError(msg, stderr, err)
	}
	if !exists {
		stderr := bytes.NewBufferString(fmt.Sprintf(
			"Make sure that branch '%v' exists and run init again.", config.MasterBranch))
		err := fmt.Errorf("branch '%v' not found", config.MasterBranch)
		return errs.NewError(msg, stderr, err)
	}

	// Make sure that the trunk branch exists.
	msg = "Make sure the trunk branch exists"
	log.Run(msg)
	exists, stderr, err = git.RefExists(config.TrunkBranch)
	if err != nil {
		return errs.NewError(msg, stderr, err)
	}
	if !exists {
		msg := "Create the trunk branch"
		log.Log(fmt.Sprintf(
			"No branch '%s' found. Will create one for you for free!", config.TrunkBranch))
		log.NewLine(fmt.Sprintf(
			"The newly created branch is pointing to '%v'.", config.MasterBranch))
		stderr, err := git.Branch(config.TrunkBranch, config.MasterBranch)
		if err != nil {
			return errs.NewError(msg, stderr, err)
		}

		msg = "Push the newly created trunk branch"
		log.Run(msg)
		_, stderr, err = git.Git("push", "-u", config.OriginName,
			config.TrunkBranch+":"+config.TrunkBranch)
		if err != nil {
			return errs.NewError(msg, stderr, err)
		}
	}

	// Check the global configuration file.
	msg = "Check the global SalsaFlow configuration"
	log.Run(msg)
	if _, err := config.ReadGlobalConfig(); err != nil {
		return errs.NewError(msg, nil,
			fmt.Errorf("could not read config file '%v': %v",
				"$HOME/"+config.GlobalConfigFileName, err))
	}

	// Check the project-specific configuration file.
	msg = "Check the local SalsaFlow configuration"
	log.Run(msg)
	if _, stderr, err = config.ReadLocalConfig(); err != nil {
		return errs.NewError(msg, stderr,
			fmt.Errorf("could not read config file '%v' on branch '%v': %v",
				config.LocalConfigFileName, config.ConfigBranch, err))
	}

	// Verify our git hook is installed and used.
	msg = "Check the git commit-msg hook"
	log.Run(msg)
	if err := checkGitHook(); err != nil {
		return errs.NewError(msg, nil, err)
	}

	// Success! Mark the repository as initialized in git config.
	msg = "Mark the repository as initialized"
	if stderr, err := git.SetConfigBool("salsaflow.initialized", true); err != nil {
		return errs.NewError(msg, stderr, err)
	}
	asciiart.PrintThumbsUp()
	fmt.Println()
	log.Log("The repository is initialised")

	return nil
}

// Check whether SalsaFlow git hook is used. Prompts user to install our hook if it isn't.
func checkGitHook() *errs.Error {
	// Ping the git hook with our secret argument.
	msg := "Get the repository root absolute path"
	repoRoot, _, err := git.RepositoryRootAbsolutePath()
	if err != nil {
		return errs.NewError(msg, nil, err)
	}

	hookPath := filepath.Join(repoRoot, ".git", "hooks", "commit-msg")
	stdout, _, _ := shell.Run(hookPath, config.SecretGitHookFilename)
	secret := strings.TrimSpace(stdout.String())

	if secret == config.SecretGitHookResponse {
		return nil
	}

	// Prompt the user to confirm the SalsaFlow git commit-msg hook.
	log.Warn("SalsaFlow git commit-msg hook not detected")
	msg = "Prompt the user to confirm the commit-msg hook"

	// Get the hook executable absolute path. It's supposed to be installed
	// in the same directory as the salsaflow executable itself.
	binDir, err := osext.ExecutableFolder()
	if err != nil {
		return errs.NewError(msg, nil, err)
	}
	hookBin := filepath.Join(binDir, getCommitMsgHookFileName())

	confirmed, err := prompt.Confirm(`
I need my own git commit-msg hook to be placed in the repository.
Shall I create or replace your current commit-msg hook?`)
	fmt.Println()
	if err != nil {
		return errs.NewError(msg, nil, err)
	}
	if !confirmed {
		// User stubbornly refuses to let us overwrite their webhook.
		// Inform the init has failed and let them do their thing.
		fmt.Printf(`I need the hook in order to do my job!

Please make sure the executable located at

  %v

runs as your commit-msg hook and run me again!

`, hookBin)
		return errs.NewError(msg, nil, errors.New("SalsaFlow git commit-msg hook not detected"))
	}

	// Install the SalsaFlow commit-msg git hook by copying the hook executable
	// from the expected absolute path to the git config hook directory.
	msg = "Install the SalsaFlow git commit-msg hook"
	if err := CopyFile(hookBin, hookPath); err != nil {
		return errs.NewError(msg, nil, err)
	}
	log.Log("SalsaFlow commit-msg git hook installed")

	return nil
}