package initCmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"bitbucket.org/kardianos/osext"
	"github.com/salsita/SalsaFlow/git-trunk/app"
	"github.com/salsita/SalsaFlow/git-trunk/asciiart"
	"github.com/salsita/SalsaFlow/git-trunk/config"
	"github.com/salsita/SalsaFlow/git-trunk/git"
	"github.com/salsita/SalsaFlow/git-trunk/log"
	"github.com/salsita/SalsaFlow/git-trunk/prompt"
	"github.com/salsita/SalsaFlow/git-trunk/shell"

	"gopkg.in/tchap/gocli.v1"
)

var Command = &gocli.Command{
	UsageLine: "init",
	Short:     "Initializes repository",
	Long: `
  Initializes repository so that it works with SalsaFlow.
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

// Expected init error.
type errorWithInfo struct {
	// What happened.
	error string
	// How to fix it.
	info string
}

// Print (formatted) expected error with advice on how to fix it.
func (err errorWithInfo) Print() {
	log.Fail(err.error)
	log.Println(err.info + "\n")
}

func runMain() (fatalErr error) {
	var (
		success bool = true
		stderr  *bytes.Buffer
		err     error
	)

	// Handles expected init errors or success (i.e., no expected errors).
	defer func() {
		if !success {
			return
		}

		// Success! Let's deal with it.
		// Set an `initialized` flag in local git config.
		_, stderr, err = git.Git("config", "salsaflow.initialized", "true")
		if err != nil {
			log.FailWithDetails(err.Error(), stderr)
			return
		}
		asciiart.PrintThumbsUp()
		log.Println("\nSwell! We have initialized your repo for great good.")
	}()

	// Handles unexpected error.
	defer func() {
		if fatalErr != nil {
			log.FailWithDetails(fatalErr.Error(), stderr)
			log.Fatal()
		}
	}()

	// Check git branches.
	var branchExists bool

	log.Run("Checking git branches.")

	branchExists, stderr, fatalErr = git.RefExists(config.MasterBranch)
	if fatalErr != nil {
		return
	}
	if !branchExists {
		log.Fail(fmt.Sprintf("Branch %s not detected", config.MasterBranch))
		log.Fatalln(fmt.Sprintf("I need branch '%s' to exist. Please make sure there "+
			"is one and run me again!", config.MasterBranch))
	}

	branchExists, stderr, fatalErr = git.RefExists(config.TrunkBranch)
	if fatalErr != nil {
		return
	}
	if !branchExists {
		log.Run(fmt.Sprintf("No branch %s found. Will create one for you for free!",
			config.TrunkBranch))
		stderr, fatalErr = git.Branch(config.TrunkBranch, config.MasterBranch)
		if fatalErr != nil {
			// TODO
			return
		}
	}

	log.Run("Checking config files.")
	if cfgErr := config.Load(); cfgErr != nil {
		logger := log.V(log.Info)
		cfgErr.Log(logger)
		success = false
	}

	if !success {
		log.Println(fmt.Sprintf(`So listen, here's how it works. We expect you to have two files: '${HOME}/%s' for storing global configuration and '${YOUR_PROJECT_ROOT_DIR}/%s' for storing project configuration. Please refer to https://github.com/salsita/SalsaFlow to find what should be in those files.`, config.GlobalConfigFileName, config.LocalConfigFileName))
		return
	}

	// Verify our git hook is installed and used.
	log.Run("Checking git hook.")
	if fatalErr, _success := checkGitHook(); fatalErr != nil {
		success = _success && success
		return fatalErr
	}

	log.Run("Taking off every zig.")

	return
}

// Check whether SalsaFlow git hook is used. Prompts user to install our hook if it
// isn't.
func checkGitHook() (fatalErr error, success bool) {
	success = false

	// Handles unexpected error.
	defer func() {
		if fatalErr != nil {
			log.FailWithDetails(fatalErr.Error(), nil)
			log.Fatal()
		}
	}()

	repoRoot, _, fatalErr := git.RepositoryRootAbsolutePath()
	if fatalErr != nil {
		return
	}

	hookPath := filepath.Join(repoRoot, ".git", "hooks", "commit-msg")
	// Ping the git hook with our secret argument.
	stdout, _, _ := shell.Run(hookPath, config.SecretGitHookFilename)
	secret := strings.TrimSpace(stdout.String())

	// Check if we got the expected response.
	if secret != config.SecretGitHookResponse {
		var confirmed bool
		log.Warn("We did not detect our webhook in your repo.")
		log.Println("")
		confirmed, fatalErr = prompt.Confirm("Shall we proceed and create or replace " +
			"your current commit-msg hook with our GitFlow hook?")
		fmt.Println("")
		if fatalErr != nil {
			return
		}
		// Get the directory our executable is in (we expect the git hook binary to be
		// installed in the same directory).
		var binDir string
		binDir, fatalErr = osext.ExecutableFolder()
		if fatalErr != nil {
			return
		}
		hookBin := filepath.Join(binDir, "git-trunk-hooks-commit-msg")
		if confirmed {
			// Copy the git hook binary to the githook directory.
			fatalErr = CopyFile(filepath.Join(binDir, "git-trunk-hooks-commit-msg"), hookPath)
			if fatalErr != nil {
				return
			}
			log.Run("Hook installed. Sweet.")
		} else {
			// User stubbornly refuses to let us overwrite their webhook. Inform the init
			// has failed and let them do their thing.
			errorWithInfo{
				"Our commit-msg webhook not detected.",
				fmt.Sprintf("I need the hook in order to do my job. Please make "+
					"sure file %s runs as your commit-msg hook and run me again!", hookBin),
			}.Print()
			return nil, false
		}
	}

	success = true

	return
}
