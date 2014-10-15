package repo

import (
	// Stdlib
	"bytes"
	"errors"
	"fmt"
	"os"
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

type hookType string

const (
	hookTypeCommitMsg hookType = "commit-msg"
	hookTypePrePush            = "pre-push"
)

const HookPrefix = "salsaflow-"

func getHookFileName(typ hookType) string {
	if runtime.GOOS == "windows" {
		return HookPrefix + string(typ) + ".exe"
	} else {
		return HookPrefix + string(typ)
	}
}

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

func Init() *errs.Error {
	// Check whether the repository has been initialised yet.
	msg := "Check whether the repository has been initialised"
	initialised, stderr, err := git.GetConfigBool("salsaflow.initialised")
	if err != nil {
		return errs.NewError(msg, stderr, err)
	}
	if initialised {
		return errs.NewError(msg, nil, ErrInitialised)
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

	// Verify our git hooks are installed and used.
	msg = "Check the current git commit-msg hook"
	log.Run(msg)
	if err := checkGitHook(hookTypeCommitMsg); err != nil {
		return errs.NewError(msg, nil, err)
	}

	msg = "Check the current git pre-push hook"
	log.Run(msg)
	if err := checkGitHook(hookTypePrePush); err != nil {
		return errs.NewError(msg, nil, err)
	}

	// Run other registered init hooks.
	msg = "Running the registered repository init hooks"
	log.Log(msg)
	if err := executeInitHooks(); err != nil {
		return errs.NewError(msg, nil, err)
	}

	// Success! Mark the repository as initialised in git config.
	msg = "Mark the repository as initialised"
	if stderr, err := git.SetConfigBool("salsaflow.initialised", true); err != nil {
		return errs.NewError(msg, stderr, err)
	}
	asciiart.PrintThumbsUp()
	fmt.Println()
	log.Log("The repository is initialised")

	return nil
}

// Check whether SalsaFlow git hook is used. Prompts user to install our hook if it isn't.
func checkGitHook(typ hookType) *errs.Error {
	// Declade some variables so that we can use goto.
	var confirmed bool

	// Ping the git hook with our secret argument.
	msg := "Get the repository root absolute path"
	repoRoot, _, err := git.RepositoryRootAbsolutePath()
	if err != nil {
		return errs.NewError(msg, nil, err)
	}

	hookPath := filepath.Join(repoRoot, ".git", "hooks", string(typ))
	stdout, _, _ := shell.Run(hookPath, config.SecretGitHookFilename)
	secret := strings.TrimSpace(stdout.String())

	if secret == config.SecretGitHookResponse {
		return nil
	}

	// Get the hook executable absolute path. It's supposed to be installed
	// in the same directory as the salsaflow executable itself.
	binDir, err := osext.ExecutableFolder()
	if err != nil {
		return errs.NewError(msg, nil, err)
	}
	hookBin := filepath.Join(binDir, getHookFileName(typ))

	// Check whether there is a hook already present in the repository.
	// If there is no hook, we don't have to ask the user, we can just install the hook.
	msg = fmt.Sprintf("Check whether there is a git %v hook already installed", typ)
	if _, err := os.Stat(hookPath); err != nil {
		if os.IsNotExist(err) {
			goto CopyHook
		}
		return errs.NewError(msg, nil, err)
	}

	// Prompt the user to confirm the SalsaFlow git commit-msg hook.
	msg = fmt.Sprintf("Prompt the user to confirm the %v hook", typ)
	confirmed, err = prompt.Confirm(`
I need my own git ` + string(typ) + ` hook to be placed in the repository.
Shall I create or replace your current ` + string(typ) + ` hook?`)
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

runs as your `+string(typ)+` hook and run me again!

`, hookBin)
		return errs.NewError(msg, nil, fmt.Errorf("SalsaFlow git %v hook not detected", typ))
	}

CopyHook:
	// Install the SalsaFlow git hook by copying the hook executable
	// from the expected absolute path to the git config hook directory.
	msg = fmt.Sprintf("Install the SalsaFlow git %v hook", typ)
	if err := CopyFile(hookBin, hookPath); err != nil {
		return errs.NewError(msg, nil, err)
	}
	log.Log(fmt.Sprintf("SalsaFlow git %v hook installed", typ))

	return nil
}
