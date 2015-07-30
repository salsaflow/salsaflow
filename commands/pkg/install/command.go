package installCmd

import (
	// Stdlib
	"fmt"
	"os"

	// Internal
	"github.com/salsaflow/salsaflow/app/appflags"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/pkg"
	"github.com/salsaflow/salsaflow/version"

	// Other
	"gopkg.in/tchap/gocli.v2"
)

var Command = &gocli.Command{
	UsageLine: "install [-github_owner=OWNER] [-github_repo=REPO] VERSION",
	Short:     "install chosen SalsaFlow version",
	Long: `
  Install SalsaFlow of the given version.

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
	if len(args) != 1 {
		cmd.Usage()
		os.Exit(2)
	}

	if err := runMain(args[0]); err != nil {
		if err == pkg.ErrAborted {
			fmt.Println("\nYour wish is my command, exiting now!")
			return
		}
		errs.Fatal(err)
	}

	log.Log("SalsaFlow was installed successfully")
}

func runMain(versionString string) error {
	if _, err := version.Parse(versionString); err != nil {
		return err
	}

	return pkg.Install(versionString, &pkg.InstallOptions{flagOwner, flagRepo})
}
