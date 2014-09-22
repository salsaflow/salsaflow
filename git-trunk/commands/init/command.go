package initCmd

import (
	"bytes"
	"fmt"
	"os"
	"path"
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

	app.MustInit()

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
	log.Println(err.info)
}

func runMain() (err error) {
	var (
		expectedErrors []errorWithInfo
		stderr         *bytes.Buffer
	)

	// Handles unexpected error.
	defer func() {
		if err != nil {
			asciiart.PrintScream("Oh noes, an error happened!", "I'm bailing out.")
			log.FailWithDetails(err.Error(), stderr)
		}
	}()

	// Handles expected init errors or success (i.e., no expected errors).
	defer func() {
		if len(expectedErrors) > 0 {
			for _, err := range expectedErrors {
				err.Print()
			}
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
		log.Println("\nSwell, your repo is initialized!")
	}()

	// Check git branches.
	var branchExists bool

	branchExists, stderr, err = git.RefExists(config.MasterBranch)
	if err != nil {
		return
	}
	if !branchExists {
		info := errorWithInfo{
			fmt.Sprintf("Branch %s not detected", config.MasterBranch),
			fmt.Sprintf("I need branch %s to exist. Please make sure there is one "+
				"and run me again!", config.MasterBranch),
		}
		expectedErrors = append(expectedErrors, info)
	}

	branchExists, stderr, err = git.RefExists(config.TrunkBranch)
	if err != nil {
		return
	}
	if !branchExists {
		log.Go(fmt.Sprintf("No branch %s found. Will create one for you for free!",
			config.TrunkBranch))
		stderr, err = git.Branch(config.TrunkBranch, config.MasterBranch)
		if err != nil {
			// TODO
			return
		}
	}

	// Check config files (local and global).
	if _, _, err = config.ReadLocalConfig(); err != nil {
		info := errorWithInfo{
			error: "Local config could not be read.",
			info: fmt.Sprintf("I could not read config from file %s in branch %s",
				config.LocalConfigFileName, config.ConfigBranch),
		}
		expectedErrors = append(expectedErrors, info)
	} else {
		log.Ok("Checked local config.")
	}
	if _, err := config.ReadGlobalConfig(); err != nil {
		expectedErrors = append(expectedErrors, errorWithInfo{
			error: "Global config could not be read.",
			info: fmt.Sprintf("I could not read config from file %s.",
				config.GlobalConfigFileName),
		})
	} else {
		log.Ok("Checked global config.")
	}

	// Verify our git hook is installed and used.
	err, stderr, expectedErrors = checkGitHook()

	return nil
}

// Check whether SalsaFlow git hook is used. Prompts user to install our hook if it
// isn't.
func checkGitHook() (err error, stderr *bytes.Buffer, expectedErrors []errorWithInfo) {
	repoRoot, stderr, err := git.RepositoryRootAbsolutePath()
	if err != nil {
		return
	}

	hookPath := path.Join(repoRoot, ".git", "hooks", "commit-msg")
	// Ping the git hook with our secret argument.
	stdout, _, _ := shell.Run(hookPath, config.SecretGitHookFilename)
	secret := strings.TrimSpace(stdout.String())

	// Check if we got the expected response.
	if secret != config.SecretGitHookResponse {
		var confirmed bool
		log.Warn("We did not detect our webhook in your repo.")
		log.Println("")
		confirmed, err = prompt.Confirm("Shall we proceed and create or replace " +
			"your current commit-msg hook with our GitFlow hook?")
		fmt.Println("")
		if err != nil {
			return
		}
		// Get the directory our executable is in (we expect the git hook binary to be
		// installed in the same directory).
		var binDir string
		binDir, err = osext.ExecutableFolder()
		if err != nil {
			return
		}
		hookBin := path.Join(binDir, "git-trunk-hooks-commit-msg")
		if confirmed {
			// Copy the git hook binary to the githook directory.
			err = CopyFile(path.Join(binDir, "git-trunk-hooks-commit-msg"), hookPath)
			if err != nil {
				return
			}
			log.Ok("Sweet, hook installed!")
		} else {
			// User stubbornly refuses to let us overwrite their webhook. Inform the init
			// has failed and let them do their thing.
			expectedErrors = append(expectedErrors, errorWithInfo{
				error: "Our commit-msg webhook not detected.",
				info: fmt.Sprintf("I need the hook in order to do my job. Please make "+
					"sure file %s runs as your commit-msg hook and run me again!", hookBin),
			})
			return
		}
	} else {
		log.Ok("Checked git hook.")
	}

	return nil, nil, expectedErrors
}
