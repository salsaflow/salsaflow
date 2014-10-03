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
	"github.com/salsita/salsaflow/config"
	"github.com/salsita/salsaflow/flag"
	"github.com/salsita/salsaflow/git"
	"github.com/salsita/salsaflow/log"
	"github.com/salsita/salsaflow/modules"
	"github.com/salsita/salsaflow/version"

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
			log.FailWithDetails(msg, stderr)
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
	releaseVersion, stderr, err := version.ReadFromBranch(config.ReleaseBranch)
	if err != nil {
		return
	}

	// Get the stories associated with the current release.
	msg = "Fetch stories from the issue tracker"
	if !porcelain {
		log.Run(msg)
	}
	release, err := modules.GetIssueTracker().RunningRelease(releaseVersion)
	if err != nil {
		return
	}
	stories, err := release.Stories()
	if err != nil {
		return
	}

	// Just return in case there are no relevant stories found.
	if len(stories) == 0 {
		msg = ""
		fmt.Println("\nNo relevant stories found, exiting...")
		return
	}

	// Get the list of all relevant story commits.
	msg = "Collect the relevant commits"
	if !porcelain {
		log.Run(msg)
	}
	var (
		commits     []*git.Commit
		storyGroups []*changes.StoryChangeGroup
	)
	for _, story := range stories {
		id := story.Id()
		commits, stderr, err = git.ListStoryCommits(id)
		if err != nil {
			return
		}
		if len(commits) == 0 {
			continue
		}
		storyGroups = append(storyGroups, &changes.StoryChangeGroup{
			StoryId: id,
			Changes: changes.GroupCommitsByChangeId(commits),
		})
	}

	// Just return in case there are no relevant commits found.
	if len(storyGroups) == 0 {
		fmt.Println("\nNo relevant commits found, exiting...")
		return
	}

	// Dump the change details into the console.
	tw := tabwriter.NewWriter(os.Stdout, 0, 8, 2, '\t', 0)

	if !porcelain {
		io.WriteString(tw, "\n")
		io.WriteString(tw, "Story\tChange\tCommit SHA\tCommit Source\tCommit Title\n")
		io.WriteString(tw, "=====\t======\t==========\t=============\t============\n")
	}
	for _, group := range storyGroups {
		storyId := group.StoryId

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
