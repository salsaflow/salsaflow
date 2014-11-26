package changesCmd

import (
	// Stdlib
	"errors"
	"fmt"
	"os"

	// Internal
	"github.com/salsaflow/salsaflow/app"
	"github.com/salsaflow/salsaflow/changes"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/flag"
	"github.com/salsaflow/salsaflow/git"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/modules"
	"github.com/salsaflow/salsaflow/version"

	// Other
	"gopkg.in/tchap/gocli.v1"
)

var Command = &gocli.Command{
	UsageLine: `
  changes [-porcelain]
          [-include_source=REGEXP ...]
          [-exclude_source=REGEXP ...]`,
	Short: "list the changes associated with the current release",
	Long: `
  List the change sets (the commits with the same change ID)
  associated with the current release together with some details,
  e.g. the commit SHA, the source ref and the commit title.

  -include_source and -exclude_source flags can be used to limit
  what change sets are listed. When these flags are used, every change set
  is checked and the whole set is printed iff there is a commit with the source
  matching one of the include filters and there is no commit with the source
  matching one of the exclude filters.

  -porcelain flag will make the output more script-friendly,
  e.g. it will fill the change ID in every column.
	`,
	Action: run,
}

var (
	porcelain bool
	include   = flag.NewRegexpSetFlag()
	exclude   = flag.NewRegexpSetFlag()
)

func init() {
	Command.Flags.BoolVar(&porcelain, "porcelain", false, "enable script-friendly output")
	Command.Flags.Var(include, "include_source", "source ref to include")
	Command.Flags.Var(exclude, "exclude_source", "source ref to exclude")
}

func run(cmd *gocli.Command, args []string) {
	if len(args) != 0 {
		cmd.Usage()
		os.Exit(2)
	}

	app.InitOrDie()

	log.Disable()
	err := runMain()
	log.Replace(os.Stderr)
	if err != nil {
		if porcelain {
			os.Exit(1)
		}
		errs.Fatal(err)
	}
}

func runMain() error {
	// Load repo config.
	gitConfig, err := git.LoadConfig()
	if err != nil {
		return err
	}

	var (
		remoteName    = gitConfig.RemoteName()
		releaseBranch = gitConfig.ReleaseBranchName()
	)

	// Make sure that the local release branch exists.
	task := "Make sure that the local release branch exists"
	err = git.CreateTrackingBranchUnlessExists(releaseBranch, remoteName)
	if err != nil {
		return errs.NewError(task, err, nil)
	}

	// Get the current release version string.
	task = "Get the current release version string"
	releaseVersion, err := version.GetByBranch(releaseBranch)
	if err != nil {
		return errs.NewError(task, err, nil)
	}

	// Get the stories associated with the current release.
	task = "Get the stories associated with the current release"
	tracker, err := modules.GetIssueTracker()
	if err != nil {
		return errs.NewError(task, err, nil)
	}
	release, err := tracker.RunningRelease(releaseVersion)
	if err != nil {
		return errs.NewError(task, err, nil)
	}
	stories, err := release.Stories()
	if err != nil {
		return errs.NewError(task, err, nil)
	}

	if len(stories) == 0 {
		return errs.NewError(task, errors.New("no relevant stories found"), nil)
	}

	// Get the story changes.
	task = "Collect the story changes"
	log.Run(task)
	storyGroups, err := changes.StoryChanges(stories, include.Values, exclude.Values)
	if err != nil {
		return errs.NewError(task, err, nil)
	}

	// Just return in case there are no relevant commits found.
	if len(storyGroups) == 0 {
		return errs.NewError(task, errors.New("no relevant commits found"), nil)
	}

	// Dump the change details into the console.
	if !porcelain {
		fmt.Println()
	}
	changes.DumpStoryChanges(storyGroups, os.Stdout, porcelain)
	if !porcelain {
		fmt.Println()
	}

	return nil
}
