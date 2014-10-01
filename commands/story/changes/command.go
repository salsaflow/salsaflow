package changesCmd

import (
	// Stdlib
	"bytes"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	// Internal
	"github.com/salsita/salsaflow/app"
	"github.com/salsita/salsaflow/changes"
	"github.com/salsita/salsaflow/flag"
	"github.com/salsita/salsaflow/git"
	"github.com/salsita/salsaflow/log"

	// Other
	"gopkg.in/tchap/gocli.v1"
)

var Command = &gocli.Command{
	UsageLine: `
  changes [-porcelain]
          [-include_source=REGEXP ...]
          [-exclude_source=REGEXP ...]
          STORY`,
	Short: "list the changes associated with the given story",
	Long: `
  List the change sets (the commits with the same change ID)
  associated with the given story together with some interesting details,
  e.g. the commit SHA, the source ref and the commit title.

  -include_source and -exclude_source flags can be used to limit
  what change sets are actually listed. When these flags are used,
  every change set (the commits with the same change ID) is checked
  and the whole set is printed iff there is a commit with the source
  matching one of the include filters and there is no commit with
  the source matching one of the exclude filters.

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
	if len(args) != 1 {
		cmd.Usage()
		os.Exit(2)
	}

	app.MustInit()

	if err := runMain(args[0]); err != nil {
		log.Fatalln("\nError: " + err.Error())
	}
}

func runMain(storyId string) (err error) {
	var (
		msg    string
		stderr *bytes.Buffer
	)
	defer func() {
		// Print error details.
		if err != nil {
			log.FailWithDetails(msg, stderr)
		}
	}()

	// Get the list of all relevant story commits.
	msg = "Get the list of relevant story commits"
	commits, stderr, err := git.ListStoryCommits(storyId)
	if err != nil {
		return
	}

	// Group the commits by change ID.
	groups := changes.GroupCommitsByChangeId(commits)
	groups = changes.FilterChangesBySource(groups, include.Values, exclude.Values)

	// Dump the change details into the console.
	tw := tabwriter.NewWriter(os.Stdout, 0, 8, 2, '\t', 0)

	if !porcelain {
		io.WriteString(tw, "\n")
		io.WriteString(tw, "Change\tCommit SHA\tCommit Source\tCommit Title\n")
		io.WriteString(tw, "======\t==========\t=============\t============\n")
	}
	for _, group := range groups {
		commit := group.Commits[0]
		var changeId string
		if porcelain {
			changeId = commit.ChangeId
		}

		fmt.Fprintf(tw, "%v\t%v\t%v\t%v\n",
			commit.ChangeId, commit.SHA, commit.Source, commit.Title)
		for _, commit := range group.Commits[1:] {
			fmt.Fprintf(tw, "%v\t%v\t%v\t%v\n", changeId, commit.SHA, commit.Source, commit.Title)
		}
	}
	if !porcelain {
		io.WriteString(tw, "\n")
	}

	tw.Flush()
	return nil
}
