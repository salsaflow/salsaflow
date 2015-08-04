package upgradeCmd

import (
	// Stdlib
	"fmt"
	"os"

	// Internal
	"github.com/salsaflow/salsaflow/app/appflags"
	"github.com/salsaflow/salsaflow/asciiart"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/pkg"

	// Other
	"gopkg.in/tchap/gocli.v2"
)

var Command = &gocli.Command{
	UsageLine: "upgrade [-github_owner=OWNER] [-github_repo=REPO]",
	Short:     "upgrade SalsaFlow executables",
	Long: `
  Upgrade SalsaFlow executables to the most recent version.

  The default GitHub repository to be used to fetch SalsaFlow releases
  can be overwritten using the available command line flags.
	`,
	Action: run,
}

var (
	flagOwner = pkg.DefaultGitHubOwner
	flagRepo  = pkg.DefaultGitHubRepo
)

func init() {
	// Register flags.
	Command.Flags.StringVar(&flagOwner, "github_owner", flagOwner, "GitHub account name")
	Command.Flags.StringVar(&flagRepo, "github_repo", flagRepo, "GitHub repository name")

	// Register global flags.
	appflags.RegisterGlobalFlags(&Command.Flags)
}

func run(cmd *gocli.Command, args []string) {
	if len(args) != 0 {
		cmd.Usage()
		os.Exit(2)
	}

	upgraded, err := pkg.Upgrade(&pkg.InstallOptions{
		GitHubOwner: flagOwner,
		GitHubRepo:  flagRepo,
	})
	if err != nil {
		if err == pkg.ErrAborted {
			fmt.Println("\nYour wish is my command, exiting now!")
			return
		}
		errs.Fatal(err)
	}

	if upgraded {
		log.Log("SalsaFlow was upgraded successfully")
	} else {
		log.Log("SalsaFlow is up to date")
		asciiart.PrintThumbsUp()
		fmt.Println()
	}
}
