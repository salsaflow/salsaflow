package changesCmd

import (
	// Stdlib
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"text/tabwriter"

	// Internal
	"github.com/salsita/SalsaFlow/git-trunk/app"
	"github.com/salsita/SalsaFlow/git-trunk/changes"
	"github.com/salsita/SalsaFlow/git-trunk/config"
	"github.com/salsita/SalsaFlow/git-trunk/flag"
	"github.com/salsita/SalsaFlow/git-trunk/git"
	"github.com/salsita/SalsaFlow/git-trunk/log"
	"github.com/salsita/SalsaFlow/git-trunk/utils/pivotaltracker"
	"github.com/salsita/SalsaFlow/git-trunk/version"

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

	app.MustInit()

	if err := runMain(); err != nil {
		if porcelain {
			os.Exit(1)
		}
		log.Fatalln("\nError: " + err.Error())
	}
}

func runMain() (err error) {
	var (
		msg    string
		stderr *bytes.Buffer
	)
	defer func() {
		// Print error details.
		if err != nil && !porcelain {
			log.FailWithContext(msg, stderr)
		}
	}()

	// Make sure that the local release branch exists.
	msg = "Make sure that the local release branch exists"
	stderr, err = git.CreateTrackingBranchUnlessExists(config.ReleaseBranch, config.OriginName)
	if err != nil {
		return
	}

	// Get the current release version string.
	msg = "Get the current release version string"
	ver, stderr, err := version.ReadFromBranch(config.ReleaseBranch)
	if err != nil {
		return
	}

	// Get the stories associated with the current release.
	msg = "Fetch Pivotal Tracker stories"
	if !porcelain {
		log.Run(msg)
	}
	stories, err := pivotaltracker.ListStories("label:release-" + ver.String())
	if err != nil {
		return
	}

	// Just return in case there are no relevant stories found.
	if len(stories) == 0 {
		msg = ""
		err = errors.New("no relevant stories found")
		return
	}

	// Get the list of all relevant story commits.
	msg = "Get the list of relevant commits"
	var (
		cs      []*git.Commit
		commits = make([]*git.Commit, 0, len(stories))
	)
	for _, story := range stories {
		cs, stderr, err = git.ListStoryCommits(story.Id)
		if err != nil {
			return
		}
		commits = append(commits, cs...)
	}

	// Just return in case there are no relevant commits found.
	if len(commits) == 0 {
		err = errors.New("no relevant story commits found")
		return
	}

	// Create the story change groups.
	storyGroups := changes.GroupChangesByStoryId(changes.GroupCommitsByChangeId(commits))

	// Dump the change details into the console.
	tw := tabwriter.NewWriter(os.Stdout, 0, 8, 2, '\t', 0)

	if !porcelain {
		io.WriteString(tw, "\n")
		io.WriteString(tw, "Story\tChange\tCommit SHA\tCommit Source\tCommit Title\n")
		io.WriteString(tw, "=====\t======\t==========\t=============\t============\n")
	}
	for _, group := range storyGroups {
		storyId := strconv.Itoa(group.StoryId)

		for _, change := range group.Changes {
			changeId := change.ChangeId

			// Print the first line.
			commit := change.Commits[0]
			fmt.Fprintf(tw, "%v\t%v\t%v\t%v\t%v\n",
				storyId, changeId, commit.SHA, commit.Source, commit.Title)

			// Make some of the columns empty in case we are not porcelain.
			if !porcelain {
				storyId = ""
				changeId = ""
			}

			// Print the rest with the chosen columns being empty.
			for _, commit := range change.Commits[1:] {
				fmt.Fprintf(tw, "%v\t%v\t%v\t%v\t%v\n",
					storyId, changeId, commit.SHA, commit.Source, commit.Title)
			}
		}
	}
	if !porcelain {
		io.WriteString(tw, "\n")
	}

	tw.Flush()
	return nil
}
