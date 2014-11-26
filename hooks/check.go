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
	"github.com/salsaflow/salsaflow/git/gitutil"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/prompt"
	"github.com/salsaflow/salsaflow/shell"

	// Other
	"bitbucket.org/kardianos/osext"
)

type HookType string

const (
	HookTypeCommitMsg HookType = "commit-msg"
	HookTypePrePush            = "pre-push"
)

const hookPrefix = "salsaflow-"

func getHookFileName(typ HookType) string {
	if runtime.GOOS == "windows" {
		return hookPrefix + string(typ) + ".exe"
	} else {
		return hookPrefix + string(typ)
	}
}

// Check whether SalsaFlow git hook is used. Prompts user to install our hook if it isn't.
func CheckAndUpsert(typ HookType) error {
	// Declade some variables so that we can use goto.
	var confirmed bool

	// Ping the git hook with our secret argument.
	repoRoot, err := gitutil.RepositoryRootAbsolutePath()
	if err != nil {
		return err
	}

	hookPath := filepath.Join(repoRoot, ".git", "hooks", string(typ))
	stdout, _, _ := shell.Run(hookPath, "-"+versionFlag)

	// In case the versions match, we are done here.
	if strings.TrimSpace(stdout.String()) == metadata.Version {
		return nil
	}

	// Get the hook executable absolute path. It's supposed to be installed
	// in the same directory as the salsaflow executable itself.
	task := "Get the executable folder absolute path"
	binDir, err := osext.ExecutableFolder()
	if err != nil {
		return errs.NewError(task, err, nil)
	}
	hookBin := filepath.Join(binDir, getHookFileName(typ))

	// Check whether there is a hook already present in the repository.
	// If there is no hook, we don't have to ask the user, we can just install the hook.
	task = fmt.Sprintf("Check whether there is a git %v hook already installed", typ)
	if _, err := os.Stat(hookPath); err != nil {
		if os.IsNotExist(err) {
			goto CopyHook
		}
		return errs.NewError(task, err, nil)
	}

	// Prompt the user to confirm the SalsaFlow git commit-task hook.
	task = fmt.Sprintf("Prompt the user to confirm the %v hook", typ)
	confirmed, err = prompt.Confirm(`
I need my own git ` + string(typ) + ` hook to be placed in the repository.
Shall I create or replace your current ` + string(typ) + ` hook?`)
	fmt.Println()
	if err != nil {
		return errs.NewError(task, err, nil)
	}
	if !confirmed {
		// User stubbornly refuses to let us overwrite their webhook.
		// Inform the init has failed and let them do their thing.
		fmt.Printf(`I need the hook in order to do my job!

Please make sure the executable located at

  %v

runs as your `+string(typ)+` hook and run me again!

`, hookBin)
		return errs.NewError(task, fmt.Errorf("SalsaFlow git %v hook not detected", typ), nil)
	}

CopyHook:
	// Install the SalsaFlow git hook by copying the hook executable
	// from the expected absolute path to the git config hook directory.
	task = fmt.Sprintf("Install the SalsaFlow git %v hook", typ)
	if err := CopyFile(hookBin, hookPath); err != nil {
		return errs.NewError(task, err, nil)
	}
	log.Log(fmt.Sprintf("SalsaFlow git %v hook installed", typ))

	return nil
}
