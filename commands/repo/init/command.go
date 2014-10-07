package initCmd

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
	"github.com/salsita/salsaflow/app"
	"github.com/salsita/salsaflow/asciiart"
	"github.com/salsita/salsaflow/config"
	"github.com/salsita/salsaflow/errs"
	"github.com/salsita/salsaflow/git"
	"github.com/salsita/salsaflow/log"
	"github.com/salsita/salsaflow/prompt"
	"github.com/salsita/salsaflow/shell"

	// Other
	"bitbucket.org/kardianos/osext"
	"gopkg.in/tchap/gocli.v1"
)

var CommitMsgHookFileName = "salsaflow-commit-msg"

func init() {
	if runtime.GOOS == "windows" {
		CommitMsgHookFileName += ".exe"
	}
}

var Command = &gocli.Command{
	UsageLine: "init",
	Short:     "initialize the repository",
	Long: `
  Initialize the repository so that it works with SalsaFlow.
	`,
	Action: run,
}

func run(cmd *gocli.Command, args []string) {
	if len(args) != 0 {
		cmd.Usage()
		os.Exit(2)
	}

	// Ignore errors here (we'll sort them out in runMain).
	app.Init()

	if err := runMain(); err != nil {
		log.Fatalln("\nError: " + err.Error())
	}
}

func handleError(task string, err error, stderr *bytes.Buffer) error {
	errs.NewError(task, stderr, err).Log(log.V(log.Info))
	return err
}

func runMain() (err error) {
	// Handles expected init errors or success (i.e., no expected errors).
	defer func() {
		if err != nil {
			return
		}

		// Success! Mark the repository as initialized in git config.
		msg := "Mark the repository as initialized"
		_, stderr, ex := git.Git("config", "salsaflow.initialized", "true")
		if ex != nil {
			err = handleError(msg, ex, stderr)
			return
		}
		asciiart.PrintThumbsUp()
		log.Println("\nSwell, your repo is initialized!\n")
	}()

	// Check whether the repository has been initialized yet.
	msg := "Check whether the repository has been initialized yet"
	initialized, stderr, err := git.GetConfigBool("salsaflow.initialized")
	if err != nil {
		return handleError(msg, err, stderr)
	}
	if initialized {
		return errors.New("repository already initialized")
	}

	// Make sure the user is using the right version of Git.
	//
	// The check is here and not in app.Init because it is highly improbable
	// that the check would pass onece and then fail later. It is expected
	// that once the user starts using git version 2.x, he keeps doing so.
	msg = "Check the git version being used"
	log.Run(msg)
	stdout, stderr, err := shell.Run("git", "--version")
	if err != nil {
		return handleError(msg, err, stderr)
	}
	pattern := regexp.MustCompile("^git version ([0-9]+)[.]([0-9]+)[.]([0-9]+)")
	parts := pattern.FindStringSubmatch(stdout.String())
	if len(parts) != 4 {
		return handleError(msg, errors.New("unexpected git --version output"), nil)
	}
	major, _ := strconv.Atoi(parts[1])
	minor, _ := strconv.Atoi(parts[2])
	patch, _ := strconv.Atoi(parts[3])
	gitVersion := fmt.Sprintf("%v.%v.%v", major, minor, patch)
	if runtime.GOOS == "windows" {
		// We require git 1.9.4+ on Windows.
		switch {
		case major >= 2:
			// OK
		case major == 1 && minor > 9:
			// OK
		case major == 1 && minor == 9 && patch >= 4:
			// OK
		default:
			return handleError(
				msg,
				errors.New("unsupported git version detected: "+gitVersion),
				nil)
		}
	} else {
		// Don't bother, just require git 2.0.0+ on other systems.
		if major < 2 {
			return handleError(
				msg,
				errors.New("unsupported git version detected: "+gitVersion),
				nil)
		}
	}

	// Make sure that the master branch exists.
	msg = "Make sure the master branch exists"
	log.Run(msg)
	exists, stderr, err := git.RefExists(config.MasterBranch)
	if err != nil {
		return handleError(msg, err, stderr)
	}
	if !exists {
		log.Fail(msg)
		log.NewLine(fmt.Sprintf(
			"Make sure that branch '%v' exists and run init again.", config.MasterBranch))
		return fmt.Errorf("branch '%v' not found", config.MasterBranch)
	}

	// Make sure that the trunk branch exists.
	msg = "Make sure the trunk branch exists"
	log.Run(msg)
	exists, stderr, err = git.RefExists(config.TrunkBranch)
	if err != nil {
		return handleError(msg, err, stderr)
	}
	if !exists {
		msg := "Create the trunk branch"
		log.Log(fmt.Sprintf(
			"No branch '%s' found. Will create one for you for free!", config.TrunkBranch))
		log.NewLine(fmt.Sprintf(
			"The newly created branch is pointing to '%v'.", config.MasterBranch))
		stderr, err := git.Branch(config.TrunkBranch, config.MasterBranch)
		if err != nil {
			return handleError(msg, err, stderr)
		}

		msg = "Push the newly created trunk branch"
		log.Run(msg)
		_, stderr, err = git.Git("push", "-u", config.OriginName,
			config.TrunkBranch+":"+config.TrunkBranch)
		if err != nil {
			return handleError(msg, err, stderr)
		}
	}

	// Check the project-specific configuration file.
	msg = "Check the local SalsaFlow configuration"
	log.Run(msg)
	if _, stderr, err = config.ReadLocalConfig(); err != nil {
		return handleError(msg, fmt.Errorf("could not read config file '%v' on branch '%v': %v",
			config.LocalConfigFileName, config.ConfigBranch, err), stderr)
	}

	// Check the global configuration file.
	msg = "Check the global SalsaFlow configuration"
	log.Run(msg)
	if _, err := config.ReadGlobalConfig(); err != nil {
		return handleError(msg, fmt.Errorf("could not read config file '%v': %v",
			"$HOME/"+config.GlobalConfigFileName, err), nil)
	}

	// Verify our git hook is installed and used.
	msg = "Check the git commit-msg hook"
	log.Run(msg)
	if err := checkGitHook(); err != nil {
		return handleError(msg, err, nil)
	}

	return nil
}

// Check whether SalsaFlow git hook is used. Prompts user to install our hook if it isn't.
func checkGitHook() error {
	// Ping the git hook with our secret argument.
	repoRoot, _, err := git.RepositoryRootAbsolutePath()
	if err != nil {
		return err
	}

	hookPath := filepath.Join(repoRoot, ".git", "hooks", "commit-msg")
	stdout, _, _ := shell.Run(hookPath, config.SecretGitHookFilename)
	secret := strings.TrimSpace(stdout.String())

	if secret == config.SecretGitHookResponse {
		return nil
	}

	// Prompt the user to confirm the SalsaFlow git commit-msg hook.
	log.Warn("SalsaFlow git commit-msg hook not detected")
	msg := "Prompt the user to confirm the commit-msg hook"

	// Get the hook executable absolute path. It's supposed to be installed
	// in the same directory as the salsaflow executable itself.
	binDir, err := osext.ExecutableFolder()
	if err != nil {
		return handleError(msg, err, nil)
	}
	hookBin := filepath.Join(binDir, CommitMsgHookFileName)

	confirmed, err := prompt.Confirm(`
I need my own git commit-msg hook to be placed in the repository.
Shall I create or replace your current commit-msg hook?`)
	fmt.Println()
	if err != nil {
		return handleError(msg, err, nil)
	}
	if !confirmed {
		// User stubbornly refuses to let us overwrite their webhook.
		// Inform the init has failed and let them do their thing.
		fmt.Printf(`I need the hook in order to do my job!

Please make sure the executable located at

  %v

runs as your commit-msg hook and run me again!

`, hookBin)
		return errors.New("SalsaFlow git commit-msg hook not detected")
	}

	// Install the SalsaFlow commit-msg git hook by copying the hook executable
	// from the expected absolute path to the git config hook directory.
	msg = "Install the SalsaFlow git commit-msg hook"
	if err := CopyFile(hookBin, hookPath); err != nil {
		return handleError(msg, err, nil)
	}
	log.Log("SalsaFlow commit-msg git hook installed. Sweet.")

	return nil
}
