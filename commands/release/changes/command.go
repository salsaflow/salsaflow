package changesCmd

import (
	// Stdlib
	"errors"
	"fmt"
	"os"

	// Internal
	"github.com/salsaflow/salsaflow/app"
	"github.com/salsaflow/salsaflow/app/appflags"
	"github.com/salsaflow/salsaflow/changes"
	"github.com/salsaflow/salsaflow/commands/release/changes/cherrypick"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/git"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/modules"
	"github.com/salsaflow/salsaflow/releases"
	"github.com/salsaflow/salsaflow/version"

	// Other
	"gopkg.in/tchap/gocli.v2"
)

var Command = &gocli.Command{
	UsageLine: "changes [-porcelain] [-to_cherrypick]",
	Short:     "list the changes associated with the current release",
	Long: `
  List the change sets (the commits with the same change ID)
  associated with the current release together with some details,
  e.g. the commit SHA, the source ref and the commit title.

  The 'porcelain' flag will make the output more script-friendly,
  e.g. it will fill the change ID in every column.

  The 'to_cherrypick' flag can be used to list the changes that are assigned
  to the release but haven't been cherry-picked onto the release branch yet.
	`,
	Action: run,
}

var (
	flagPorcelain    bool
	flagToCherryPick bool
)

func init() {
	// Register flags.
	Command.Flags.BoolVar(&flagPorcelain, "porcelain", flagPorcelain,
		"enable script-friendly output")
	Command.Flags.BoolVar(&flagToCherryPick, "to_cherrypick", flagToCherryPick,
		"list the changes to cherry-pick")

	// Register global flags.
	appflags.RegisterGlobalFlags(&Command.Flags)

	// Register the subcommands.
	Command.MustRegisterSubcommand(cherrypickCmd.Command)
}

func run(cmd *gocli.Command, args []string) {
	if len(args) != 0 {
		cmd.Usage()
		os.Exit(2)
	}

	app.InitOrDie()

	if flagPorcelain {
		log.Disable()
	}
	err := runMain()
	if err != nil {
		errs.Fatal(err)
	}
}

func runMain() (err error) {
	// Load repo config.
	gitConfig, err := git.LoadConfig()
	if err != nil {
		return err
	}

	var (
		remoteName    = gitConfig.RemoteName
		releaseBranch = gitConfig.ReleaseBranchName
	)

	// Make sure that the local release branch exists.
	task := "Make sure that the local release branch exists"
	if err := git.CheckOrCreateTrackingBranch(releaseBranch, remoteName); err != nil {
		return errs.NewError(task, err)
	}

	// Get the current release version string.
	task = "Get the release branch version string"
	releaseVersion, err := version.GetByBranch(releaseBranch)
	if err != nil {
		return errs.NewError(task, err)
	}

	// Get the stories associated with the current release.
	task = "Fetch the stories associated with the current release"
	log.Run(task)
	tracker, err := modules.GetIssueTracker()
	if err != nil {
		return errs.NewError(task, err)
	}
	release := tracker.RunningRelease(releaseVersion)
	stories, err := release.Stories()
	if err != nil {
		return errs.NewError(task, err)
	}

	if len(stories) == 0 {
		return errs.NewError(task, errors.New("no relevant stories found"))
	}

	// Get the story changes.
	task = "Collect the story changes"
	log.Run(task)
	groups, err := changes.StoryChanges(stories)
	if err != nil {
		return errs.NewError(task, err)
	}

	// Just return in case there are no relevant commits found.
	if len(groups) == 0 {
		return errs.NewError(task, errors.New("no relevant commits found"))
	}

	// Sort the change groups.
	groups = changes.SortStoryChanges(groups, stories)

	if flagToCherryPick {
		groups, err = releases.StoryChangesToCherryPick(groups)
		if err != nil {
			return errs.NewError(task, err)
		}
	}

	// Dump the change details into the console.
	if !flagPorcelain {
		fmt.Println()
	}
	changes.DumpStoryChanges(os.Stdout, groups, tracker, flagPorcelain)
	if !flagPorcelain {
		fmt.Println()
	}

	return nil
}
