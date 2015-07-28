package changesCmd

import (
	// Stdlib
	"fmt"
	"os"

	// Internal
	"github.com/salsaflow/salsaflow/app"
	"github.com/salsaflow/salsaflow/app/appflags"
	"github.com/salsaflow/salsaflow/changes"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/git"
	"github.com/salsaflow/salsaflow/modules"

	// Other
	"gopkg.in/tchap/gocli.v2"
)

var Command = &gocli.Command{
	UsageLine: "changes [-porcelain] STORY_ID_TAG_PATTERN",
	Short:     "list the changes associated with the given story",
	Long: `
  List the change sets (the commits with the same change ID)
  associated with the given stories together with some interesting details,
  e.g. the commit SHA, the source ref and the commit title.

  The changes (commits) to be included are specified using a regexp
  that is used to match the Story-Id tag, so the commits having the tag
  matching STORY_ID_TAG_PATTERN are selected and printed.

  The 'porcelain' flag will make the output more script-friendly,
  e.g. it will fill the change ID in every column.
	`,
	Action: run,
}

var (
	porcelain bool
)

func init() {
	// Register flags.
	Command.Flags.BoolVar(&porcelain, "porcelain", false, "enable script-friendly output")

	// Register global flags.
	appflags.RegisterGlobalFlags(&Command.Flags)
}

func run(cmd *gocli.Command, args []string) {
	if len(args) != 1 {
		cmd.Usage()
		os.Exit(2)
	}

	app.InitOrDie()

	if err := runMain(args[0]); err != nil {
		errs.Fatal(err)
	}
}

func runMain(storyIdPattern string) (err error) {
	// Get the issue tracker instance.
	tracker, err := modules.GetIssueTracker()
	if err != nil {
		return err
	}

	// Get the list of all relevant changes.
	task := "Get the list of relevant story commits"
	groups, err := collectChanges(storyIdPattern)
	if err != nil {
		return errs.NewError(task, err)
	}

	// Dump the changes to stdout.
	if !porcelain {
		fmt.Println()
	}
	err = changes.DumpStoryChanges(os.Stdout, groups, tracker, porcelain)
	if !porcelain {
		fmt.Println()
	}
	return err
}

func collectChanges(pattern string) ([]*changes.StoryChangeGroup, error) {
	// Collect the relevant commits.
	commits, err := git.GrepCommitsCaseInsensitive(
		fmt.Sprintf("^Story-Id: .*%v", pattern), "--all")
	if err != nil {
		return nil, err
	}

	// Group the commits.
	return changes.StoryChangesFromCommits(commits)
}
