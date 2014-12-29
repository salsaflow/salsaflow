package cherrypickCmd

import (
	// Stdlib
	"bytes"
	"errors"
	"fmt"
	"os"

	// Internal
	"github.com/salsaflow/salsaflow/app"
	"github.com/salsaflow/salsaflow/changes"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/git"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/modules"
	"github.com/salsaflow/salsaflow/prompt"
	"github.com/salsaflow/salsaflow/releases"
	"github.com/salsaflow/salsaflow/version"

	// Other
	"gopkg.in/tchap/gocli.v2"
)

var Command = &gocli.Command{
	UsageLine: "cherry-pick [-no_fetch]",
	Short:     "cherry-pick the missing commits into the release branch",
	Long: `
Take all the commits as listed by 'release changes -to_cherrypick'
and apply them to the release branch, thus synchronizing the release
branch with the trunk branch considering the release in progress.
	`,
	Action: run,
}

var flagNoFetch bool

func init() {
	Command.Flags.BoolVar(&flagNoFetch, "no_fetch", flagNoFetch,
		"do not fetch the remote repository")
}

func run(cmd *gocli.Command, args []string) {
	if len(args) != 0 {
		cmd.Usage()
		os.Exit(2)
	}

	app.InitOrDie()

	defer prompt.RecoverCancel()

	if err := runMain(); err != nil {
		errs.Fatal(err)
	}
}

func runMain() error {
	// Load git-related config.
	gitConfig, err := git.LoadConfig()
	if err != nil {
		return err
	}
	var (
		remoteName    = gitConfig.RemoteName()
		trunkBranch   = gitConfig.TrunkBranchName()
		releaseBranch = gitConfig.ReleaseBranchName()
	)

	// Fetch the remote repository unless explicitly skipped.
	if !flagNoFetch {
		task := "Fetch the remote repository"
		log.Run(task)
		if err := git.UpdateRemotes(remoteName); err != nil {
			return errs.NewError(task, err, nil)
		}
	}

	log.Run("Make sure all important branches are up to date")

	// Make sure that the local release branch exists.
	task := "Make sure that the local release branch exists"
	err = git.CreateTrackingBranchUnlessExists(releaseBranch, remoteName)
	if err != nil {
		return errs.NewError(task, err, nil)
	}

	// Make sure that the release branch is up to date.
	task = "Make sure the release branch is up to date"
	if err := git.EnsureBranchSynchronized(releaseBranch, remoteName); err != nil {
		return errs.NewError(task, err, nil)
	}

	// Make sure that the trunk branch is up to date.
	task = "Make sure the trunk branch is up to date"
	if err := git.EnsureBranchSynchronized(trunkBranch, remoteName); err != nil {
		return errs.NewError(task, err, nil)
	}

	// Remember the current branch.
	currentBranch, err := git.CurrentBranch()
	if err != nil {
		return err
	}

	// Checkout the release branch.
	task = "Checkout the release branch"
	if err := git.Checkout(releaseBranch); err != nil {
		return errs.NewError(task, err, nil)
	}
	defer func() {
		// Do not checkout the original branch in case the name is empty.
		// This is later used to disable the checkout of the original branch.
		if currentBranch == "" {
			return
		}
		// Otherwise checkout the original branch.
		task := fmt.Sprintf("Checkout the original branch (%v)", currentBranch)
		if err := git.Checkout(currentBranch); err != nil {
			errs.LogError(task, err, nil)
		}
	}()

	// Get the current release version string.
	// It is enough to just call version.Get since
	// we are already on the release branch.
	task = "Get the release branch version string"
	releaseVersion, err := version.Get()
	if err != nil {
		return errs.NewError(task, err, nil)
	}

	// Get the stories associated with the current release.
	task = "Fetch the stories associated with the current release"
	log.Run(task)
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

	// Get the release changes.
	task = "Collect the release changes"
	log.Run(task)
	groups, err := changes.StoryChanges(stories)
	if err != nil {
		return errs.NewError(task, err, nil)
	}

	// Just return in case there are no relevant commits found.
	if len(groups) == 0 {
		return errs.NewError(task, errors.New("no relevant commits found"), nil)
	}

	// Sort the change groups.
	groups = changes.SortStoryChanges(groups, stories)
	groups, err = releases.StoryChangesToCherryPick(groups)
	if err != nil {
		return errs.NewError(task, err, nil)
	}

	var (
		// Collect the changes not reachable from trunk.
		// In case there are any, we abort the cherry-picking process.
		unreachable = make([]*changes.StoryChangeGroup, 0, len(groups))

		// When we are at iterating, we also collect all release commits
		// so that we know what trunk commits to cherry-pick later.
		releaseCommits = make(map[string]struct{})

		trunkRef = fmt.Sprintf("refs/heads/%v", trunkBranch)
	)
	for _, group := range groups {
		g := &changes.StoryChangeGroup{
			StoryIdTag: group.StoryIdTag,
		}

		for _, ch := range group.Changes {
			var ok bool
			for _, c := range ch.Commits {
				// Add the commit to the map of release commits.
				releaseCommits[c.SHA] = struct{}{}

				// Look for a commit that is on trunk.
				if c.Source == trunkRef {
					ok = true
				}
			}
			if !ok {
				// In case there is none, remember the change.
				g.Changes = append(g.Changes, ch)
			}
		}

		// In case there are some changes not reachable from trunk,
		// add the story change to the list of unreachable story changes.
		if len(g.Changes) != 0 {
			unreachable = append(unreachable, g)
		}
	}

	// In case there are some changes not reachable from the trunk branch,
	// abort the process and tell the user to get the changes into trunk first.
	if len(unreachable) != 0 {
		var details bytes.Buffer
		fmt.Fprint(&details, `
The following story changes are not reachable from the trunk branch:

`)
		changes.DumpStoryChanges(&details, unreachable, tracker, false)
		fmt.Fprint(&details, `
Please cherry-pick these changes onto the trunk branch.
Only then we can proceed and cherry-pick the changes.

`)
		return errs.NewError(
			task, errors.New("commits not reachable from trunk detected"), &details)
	}

	// Everything seems fine, let's continue with the process
	// by dumping the change details into the console.
	fmt.Println()
	changes.DumpStoryChanges(os.Stdout, groups, tracker, false)

	// Ask the user to confirm before doing any cherry-picking.
	task = "Ask the user to confirm cherry-picking"
	fmt.Println(`
The changes listed above will be cherry-picked into the release branch.`)
	confirmed, err := prompt.Confirm("Are you sure you want to continue?")
	if err != nil {
		return errs.NewError(task, err, nil)
	}
	if !confirmed {
		prompt.PanicCancel()
	}
	fmt.Println()

	// Collect the trunk commits that were created since the last release.
	task = "Collect the trunk commits added since the last release"
	trunkCommits, err := releases.ListNewTrunkCommits()
	if err != nil {
		return errs.NewError(task, err, nil)
	}
	// We need the list to start with the oldest commit.
	for i, j := 0, len(trunkCommits)-1; i < j; i, j = i+1, j-1 {
		trunkCommits[i], trunkCommits[j] = trunkCommits[j], trunkCommits[i]
	}

	// Collect the commits to cherry pick. These are the commits
	// that are on trunk and they are associated with the release.
	hashesToCherryPick := make([]string, 0, len(trunkCommits))
	for _, commit := range trunkCommits {
		if _, ok := releaseCommits[commit.SHA]; ok {
			hashesToCherryPick = append(hashesToCherryPick, commit.SHA)
		}
	}

	// Perform the cherry-pick itself.
	task = "Cherry-pick the missing changes into the release branch"
	log.Run(task)
	if err := git.CherryPick(hashesToCherryPick...); err != nil {
		hint := `
It was not possible to cherry-pick the missing changes into the release branch.
The cherry-picking process might be still in progress, though. Please check
the repository status and potentially resolve the cherry-picking manually.

`
		// Do not checkout the original branch.
		currentBranch = ""
		return errs.NewError(task, err, bytes.NewBufferString(hint))
	}

	log.Log("All missing changes cherry-picked into the release branch")
	return nil
}
