package changesCmd

import (
	// Stdlib
	"bytes"
	"fmt"
	"io"
	"os"
	"sort"
	"text/tabwriter"

	// Internal
	"github.com/tchap/git-trunk/flag"
	"github.com/tchap/git-trunk/git"
	"github.com/tchap/git-trunk/log"

	// Other
	"github.com/tchap/gocli"
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

func includeGroup(cg *changeGroup) bool {
	includeMatches := false
	if len(include.Values) != 0 {
	IncludeLoop:
		for _, commit := range cg.commits {
			for _, pattern := range include.Values {
				if pattern.MatchString(commit.Source) {
					includeMatches = true
					break IncludeLoop
				}
			}
		}
	} else {
		includeMatches = true
	}

	excludeMatches := false
	if len(exclude.Values) != 0 {
	ExcludeLoop:
		for _, commit := range cg.commits {
			for _, pattern := range exclude.Values {
				if pattern.MatchString(commit.Source) {
					excludeMatches = true
					break ExcludeLoop
				}
			}
		}
	}

	return includeMatches && !excludeMatches
}

func run(cmd *gocli.Command, args []string) {
	if len(args) != 1 {
		cmd.Usage()
		os.Exit(2)
	}

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
			log.FailWithContext(msg, stderr)
		}
	}()

	// Get the list of all relevant story commits.
	msg = "Get the list of relevant story commits"
	commits, stderr, err := git.ListStoryCommits(storyId)
	if err != nil {
		return
	}

	// Group the commits by change ID.
	// The groups are internally sorted by branch significance.
	groups := groupCommitsByChangeId(commits)
	if err != nil {
		return
	}

	// Sort the groups in the list according to commit date.
	sort.Sort(changeGroups(groups))

	// Dump the change details into the console.
	tw := tabwriter.NewWriter(os.Stdout, 0, 8, 2, '\t', 0)

	if !porcelain {
		io.WriteString(tw, "\n")
		io.WriteString(tw, "Change\tCommit SHA\tCommit Source\tCommit Title\n")
		io.WriteString(tw, "======\t==========\t=============\t============\n")
	}
	for _, group := range groups {
		if !includeGroup(group) {
			continue
		}

		commit := group.commits[0]
		var changeId string
		if porcelain {
			changeId = commit.ChangeId
		}

		fmt.Fprintf(tw, "%v\t%v\t%v\t%v\n",
			commit.ChangeId, commit.SHA, commit.Source, commit.Title)
		for _, commit := range group.commits[1:] {
			fmt.Fprintf(tw, "%v\t%v\t%v\t%v\n", changeId, commit.SHA, commit.Source, commit.Title)
		}
	}
	if !porcelain {
		io.WriteString(tw, "\n")
	}

	tw.Flush()
	return nil
}

type changeGroup struct {
	commits []*git.Commit
}

func (cg *changeGroup) Add(commit *git.Commit) {
	// Just create the list in case it is empty.
	if len(cg.commits) == 0 {
		cg.commits = []*git.Commit{commit}
		return
	}

	// Insert into the sorted list of commits.
	var (
		begin int
		end   int = len(cg.commits)
		i     int
	)
	for {
		if begin == end {
			sorted := make([]*git.Commit, end, len(cg.commits)+1)
			copy(sorted, cg.commits[:end])
			sorted = append(sorted, commit)
			sorted = append(sorted, cg.commits[end:]...)
			cg.commits = sorted
			return
		}

		i = begin + (end-begin)/2
		pivot := cg.commits[i]
		if commit.CommitDate.Before(pivot.CommitDate) {
			end = i
		} else {
			begin = i + 1
		}
	}
}

func (cg *changeGroup) EarliestCommit() *git.Commit {
	if len(cg.commits) == 0 {
		return nil
	}
	return cg.commits[0]
}

func groupCommitsByChangeId(commits []*git.Commit) changeGroups {
	gs := make(map[string]*changeGroup)
	for _, commit := range commits {
		g, ok := gs[commit.ChangeId]
		if !ok {
			g = &changeGroup{}
			gs[commit.ChangeId] = g
		}
		g.Add(commit)
	}

	groups := make(changeGroups, 0, len(gs))
	for _, g := range gs {
		groups = append(groups, g)
	}
	return groups
}

// Sorting of []*changeGroup

type changeGroups []*changeGroup

func (cgs changeGroups) Len() int {
	return len(cgs)
}

func (cgs changeGroups) Less(i, j int) bool {
	return cgs[i].EarliestCommit().CommitDate.Before(cgs[j].EarliestCommit().CommitDate)
}

func (cgs changeGroups) Swap(i, j int) {
	tmp := cgs[i]
	cgs[i] = cgs[j]
	cgs[j] = tmp
}
