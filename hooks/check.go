package hooks

import (
	// Stdlib
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	// Internal
	"github.com/salsaflow/salsaflow/app/metadata"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/fileutil"
	"github.com/salsaflow/salsaflow/git/gitutil"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/prompt"
	"github.com/salsaflow/salsaflow/shell"
	"github.com/salsaflow/salsaflow/version"

	// Other
	"github.com/kardianos/osext"
)

type HookType string

const (
	HookTypeCommitMsg    HookType = "commit-msg"
	HookTypePostCheckout HookType = "post-checkout"
	HookTypePrePush      HookType = "pre-push"
)

var HookTypes = [...]HookType{
	HookTypeCommitMsg,
	HookTypePostCheckout,
	HookTypePrePush,
}

const hookPrefix = "salsaflow-"

func getHookFileName(hookType HookType) string {
	if runtime.GOOS == "windows" {
		return hookPrefix + string(hookType) + ".exe"
	} else {
		return hookPrefix + string(hookType)
	}
}

// Check whether SalsaFlow git hook is used. Prompts user to install our hook if it isn't.
//
// When the force argument is set to true, the hook is replaced when though the version matches.
func CheckAndUpsert(hookType HookType, force bool) error {
	// Declade some variables so that we can use goto.
	var confirmed bool

	// Ping the git hook with our secret argument.
	repoRoot, err := gitutil.RepositoryRootAbsolutePath()
	if err != nil {
		return err
	}

	hookDestPath := filepath.Join(repoRoot, ".git", "hooks", string(hookType))

	// Try to get the hook version.
	stdout, _, _ := shell.Run(hookDestPath, "-"+versionFlag)

	// In case the versions match, we are done here (unless force).
	installedVersion, err := version.Parse(strings.TrimSpace(stdout.String()))
	if !force && installedVersion != nil && installedVersion.String() == metadata.Version {
		return nil
	}

	// Get the hook executable absolute path. It's supposed to be installed
	// in the same directory as the salsaflow executable itself.
	task := "Get the executable folder absolute path"
	binDir, err := osext.ExecutableFolder()
	if err != nil {
		return errs.NewError(task, err)
	}
	hookExecutable := filepath.Join(binDir, getHookFileName(hookType))

	// Check whether there is a hook already present in the repository.
	// If there is no hook or there is a SalsaFlow hook returning a different version string,
	// we don't have to ask the user, we can just install the hook.
	task = fmt.Sprintf("Check whether there is a git %v hook already installed", hookType)
	if _, err := os.Stat(hookDestPath); err != nil {
		if os.IsNotExist(err) {
			return copyHook(hookType, hookExecutable, hookDestPath)
		}
		return errs.NewError(task, err)
	}
	if installedVersion != nil || force {
		return copyHook(hookType, hookExecutable, hookDestPath)
	}

	// Prompt the user to confirm the SalsaFlow git commit-task hook.
	task = fmt.Sprintf("Prompt the user to confirm the %v hook", hookType)
	confirmed, err = prompt.Confirm(`
I need my own git `+string(hookType)+` hook to be placed in the repository.
Shall I create or replace your current `+string(hookType)+` hook?`, true)
	fmt.Println()
	if err != nil {
		return errs.NewError(task, err)
	}
	if !confirmed {
		// User stubbornly refuses to let us overwrite their webhook.
		// Inform the init has failed and let them do their thing.
		fmt.Printf(`I need the hook in order to do my job!

Please make sure the executable located at

  %v

runs as your `+string(hookType)+` hook and run me again!

`, hookExecutable)
		return errs.NewError(task, fmt.Errorf("SalsaFlow git %v hook not detected", hookType))
	}

	return copyHook(hookType, hookExecutable, hookDestPath)
}

// copyHook installs the SalsaFlow git hook by copying the hook executable
// from the expected absolute path to the git config hook directory.
func copyHook(hookType HookType, hookExecutable, hookDestPath string) error {
	task := fmt.Sprintf("Install the SalsaFlow git %v hook", hookType)
	if err := fileutil.CopyFile(hookExecutable, hookDestPath); err != nil {
		return errs.NewError(task, err)
	}
	if err := os.Chmod(hookDestPath, 0750); err != nil {
		return errs.NewError(task, err)
	}
	log.Log(fmt.Sprintf("SalsaFlow git %v hook installed", hookType))
	return nil
}
