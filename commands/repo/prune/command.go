package pruneCmd

import (
	// Stdlib
	"errors"
	"fmt"
	"os"
	"strings"

	// Internal
	"github.com/salsaflow/salsaflow/app"
	"github.com/salsaflow/salsaflow/app/appflags"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/flag"
	"github.com/salsaflow/salsaflow/git"
	"github.com/salsaflow/salsaflow/git/gitutil"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/modules"
	"github.com/salsaflow/salsaflow/modules/common"
	"github.com/salsaflow/salsaflow/prompt"

	// Other
	"github.com/fatih/color"
	"gopkg.in/tchap/gocli.v2"
)

const StoryBranchPrefix = "story/"

var stateEnum = []common.StoryState{
	common.StoryStateTested,
	common.StoryStateStaged,
	common.StoryStateAccepted,
	common.StoryStateClosed,
}

var Command = &gocli.Command{
	// UsageLine set below in an init() functions.
	Short: "delete branches that are not needed",
	Long: `
  Delete Git branches that are no longer needed.

  All story branches are checked and the branches that only contain commits
  associated with stories that are in the selected state or further
  are offered to be deleted. Both local and remote branches are affected.

  This commands counts on the fact that all branches starting with story/
  are forked off trunk. In case this is not met, weird things can happen.
	`,
	Action: run,
}

var flagState *flag.StringEnumFlag

func init() {
	states := make([]string, 0, len(stateEnum))
	for _, state := range stateEnum {
		states = append(states, string(state))
	}

	// Finalize Command.
	Command.UsageLine = fmt.Sprintf("prune [-state={%v}]", strings.Join(states, "|"))

	// Register flags.
	flagState = flag.NewStringEnumFlag(states, string(common.StoryStateAccepted))
	Command.Flags.Var(flagState, "state", "set the required story state for branch removal")

	// Register global flags.
	appflags.RegisterGlobalFlags(&Command.Flags)
}

func allowedStoryStates() map[common.StoryState]struct{} {
	var enum []common.StoryState
	v := common.StoryState(flagState.Value())
	switch v {
	case common.StoryStateTested:
		enum = stateEnum
	case common.StoryStateStaged:
		enum = stateEnum[1:]
	case common.StoryStateAccepted:
		enum = stateEnum[2:]
	case common.StoryStateClosed:
		enum = stateEnum[3:]
	default:
		panic("unknown state: " + v)
	}

	m := make(map[common.StoryState]struct{}, len(enum))
	for _, state := range enum {
		m[state] = struct{}{}
	}
	return m
}

func run(cmd *gocli.Command, args []string) {
	if len(args) != 0 {
		cmd.Usage()
		os.Exit(2)
	}

	app.InitOrDie()

	if err := runMain(); err != nil {
		errs.Fatal(err)
	}
}

func runMain() error {
	// Load config.
	config, err := git.LoadConfig()
	if err != nil {
		return err
	}
	var (
		remoteName = config.RemoteName
		trunkName  = config.TrunkBranchName
	)

	// Make sure trunk is up to date.
	task := fmt.Sprintf("Make sure branch '%v' is up to date", trunkName)
	log.Run(task)
	if err := git.CheckOrCreateTrackingBranch(trunkName, remoteName); err != nil {
		return errs.NewError(task, err)
	}

	// Collect the story branches.
	task = "Collect the story branches"
	log.Run(task)
	storyBranches, err := collectStoryBranches(remoteName)
	if err != nil {
		return errs.NewError(task, err)
	}

	// Split the branches that are not up to date.
	task = "Split the branches that are not up to date"
	log.Run(task)
	storyBranches, err = splitBranchesNotInSync(storyBranches)
	if err != nil {
		return errs.NewError(task, err)
	}

	// Filter branches according to the story state.
	task = "Filter branches according to the story state"
	log.Run(task)
	filteredBranches, err := filterBranches(storyBranches, trunkName)
	if err != nil {
		return errs.NewError(task, err)
	}
	if len(storyBranches) == 0 {
		log.Log("No branches left to be deleted")
		return nil
	}

	// Prompt the user to choose what branches to delete.
	task = "Prompt the user to choose what branches to delete"
	localToDelete, remoteToDelete, err := promptUserToChooseBranches(filteredBranches)
	if err != nil {
		return errs.NewError(task, err)
	}

	// Delete chosen local branches.
	if len(localToDelete) != 0 {
		task := "Delete chosen local branches"
		log.Run(task)
		args := make([]string, 1, 1+len(localToDelete))
		args[0] = "-D"
		args = append(args, localToDelete...)
		if ex := git.Branch(args...); ex != nil {
			errs.LogError(task, ex)
			err = errors.New("failed to delete local branches")
		}
	}

	// Delete chosen remote branches.
	if len(remoteToDelete) != 0 {
		task := "Delete chosen remote branches"
		log.Run(task)
		args := make([]string, 1, 1+len(remoteToDelete))
		args[0] = "--delete"
		args = append(args, remoteToDelete...)
		if ex := git.Push(remoteName, args...); ex != nil {
			errs.LogError(task, ex)
			err = errors.New("failed to delete remote branches")
		}
	}

	return err
}

func collectStoryBranches(remoteName string) ([]*git.GitBranch, error) {
	// Load Git branches.
	branches, err := git.Branches()
	if err != nil {
		return nil, err
	}

	// Get the current branch name so that it can be excluded.
	currentBranch, err := gitutil.CurrentBranch()
	if err != nil {
		return nil, err
	}

	// Filter the branches.
	storyBranches := make([]*git.GitBranch, 0, len(branches))
	for _, branch := range branches {
		// Drop branches not corresponding to the project remote.
		if branch.Remote != "" && branch.Remote != remoteName {
			continue
		}

		var (
			isLocalStoryBranch  = strings.HasPrefix(branch.BranchName, StoryBranchPrefix)
			isRemoteStoryBranch = strings.HasPrefix(branch.RemoteBranchName, StoryBranchPrefix)
		)

		// Exclude the current branch.
		if isLocalStoryBranch && branch.BranchName == currentBranch {
			log.Warn(fmt.Sprintf("Branch '%v' is checked out, it cannot be deleted", currentBranch))
			continue
		}

		// Keep the story branches only.
		if isLocalStoryBranch || isRemoteStoryBranch {
			storyBranches = append(storyBranches, branch)
		}
	}

	// Return the result.
	return storyBranches, nil
}

func splitBranchesNotInSync(storyBranches []*git.GitBranch) ([]*git.GitBranch, error) {
	branches := make([]*git.GitBranch, 0, len(storyBranches))
	for _, branch := range storyBranches {
		upToDate, err := branch.IsUpToDate()
		if err != nil {
			return nil, err
		}
		if upToDate {
			branches = append(branches, branch)
			continue
		}

		// In case the branch is not up to date, we split the local and remote
		// reference into their own branch records to treat them separately.
		var (
			branchName       = branch.BranchName
			remoteBranchName = branch.RemoteBranchName
			remote           = branch.Remote
		)
		log.Warn(fmt.Sprintf("Branch '%s' is not up to date", branchName))
		log.NewLine(fmt.Sprintf("Treating '%v' and '%v/%v' as separate branches",
			branchName, remote, remoteBranchName))

		localBranch := &git.GitBranch{
			BranchName: branchName,
		}
		remoteBranch := &git.GitBranch{
			RemoteBranchName: remoteBranchName,
			Remote:           remote,
		}
		branches = append(branches, localBranch, remoteBranch)
	}
	return branches, nil
}

type gitBranch struct {
	tip     *git.GitBranch
	commits []*git.Commit

	// reason contains the reason the branch was included
	// in the branch deletion candidate list.
	reason string
}

func filterBranches(storyBranches []*git.GitBranch, trunkName string) ([]*gitBranch, error) {
	// Pair the branches with commit ranges specified by trunk..story
	task := "Collected commits associated with the story branches"
	branches := make([]*gitBranch, 0, len(storyBranches))
	for _, branch := range storyBranches {
		var revRange string
		if branch.BranchName != "" {
			// Handle branches that exist locally.
			revRange = fmt.Sprintf("%v..%v", trunkName, branch.BranchName)
		} else {
			// Handle branches that exist only in the remote repository.
			// We can use trunkName here since trunk is up to date.
			revRange = fmt.Sprintf("%v..%v/%v", trunkName, branch.Remote, branch.RemoteBranchName)
		}

		commits, err := git.ShowCommitRange(revRange)
		if err != nil {
			return nil, errs.NewError(task, err)
		}
		branches = append(branches, &gitBranch{
			tip:     branch,
			commits: commits,
		})
		continue
	}

	// Collect story tags.
	task = "Collect affected story tags"
	tracker, err := modules.GetIssueTracker()
	if err != nil {
		return nil, errs.NewError(task, err)
	}

	tags := make([]string, 0, len(storyBranches))
BranchLoop:
	for _, branch := range branches {
		for _, commit := range branch.commits {
			commitTag := commit.StoryIdTag

			// Make sure the tag is not in the list already.
			for _, tag := range tags {
				if tag == commitTag {
					continue BranchLoop
				}
			}

			// Drop tags not recognized by the current issue tracker.
			_, err := tracker.StoryTagToReadableStoryId(commitTag)
			if err == nil {
				tags = append(tags, commitTag)
			}
		}
	}

	// Fetch the collected stories.
	task = "Fetch associated stories from the issue tracker"
	log.Run(task)
	stories, err := tracker.ListStoriesByTag(tags)
	if err != nil {
		return nil, errs.NewError(task, err)
	}

	// Filter the branches according to the story state.
	storyByTag := make(map[string]common.Story, len(stories))
	for i, story := range stories {
		// tags[i] corresponds to stories[i]
		tag := tags[i]
		if story != nil {
			storyByTag[tag] = story
		} else {
			log.Warn(fmt.Sprintf("Story for tag '%v' was not found in the issue tracker", tag))
		}
	}

	allowedStates := allowedStoryStates()

	// checkCommits returns whether the commits passed in are ok
	// considering the state of the stories found in these commits,
	// whether the branch containing these commits can be deleted.
	checkCommits := func(commits []*git.Commit) (common.StoryState, bool) {
		var storyFound bool
		for _, commit := range commits {
			// Skip commits with empty Story-Id tag.
			if commit.StoryIdTag == "" {
				continue
			}

			// In case the story is not found, the tag is not recognized
			// by the current issue tracker. In that case we just skip the commit.
			story, ok := storyByTag[commit.StoryIdTag]
			if !ok {
				continue
			}

			// When the story state associated with the commit is not ok,
			// we can return false here to reject the branch.
			storyState := story.State()
			if _, ok := allowedStates[storyState]; !ok {
				return storyState, false
			}

			storyFound = true
		}

		// We went through all the commits and they are fine, check passed.
		return common.StoryStateInvalid, storyFound
	}

	// Go through the branches and only return these that
	// comply with the story state requirements.
	bs := make([]*gitBranch, 0, len(branches))
	for _, branch := range branches {
		tip := branch.tip

		logger := log.V(log.Verbose)
		if logger {
			logger.Log(fmt.Sprintf("Processing branch %v", tip.CanonicalName()))
		}

		// The branch can be for sure deleted in case there are no commits
		// contained in the commit range. That means the branch is merged into trunk.
		if len(branch.commits) == 0 {
			if logger {
				logger.Log("  Include the branch (reason: merged into trunk)")
			}
			branch.reason = "merged"
			bs = append(bs, branch)
			continue
		}

		// In case the commit check passed, we append the branch.
		state, ok := checkCommits(branch.commits)
		if ok {
			if logger {
				logger.Log("  Include the branch (reason: branch check passed)")
			}
			branch.reason = "check passed"
			bs = append(bs, branch)
			continue
		}

		// Otherwise we print the skip warning.
		if logger {
			if state == common.StoryStateInvalid {
				logger.Log(
					"  Exclude the branch (reason: no story commits found on the branch)")
			} else {
				logger.Log(fmt.Sprintf(
					"  Exclude the branch (reason: story state is '%v')", state))
			}
		}
	}

	return bs, nil
}

func promptUserToChooseBranches(branches []*gitBranch) (local, remote []string, err error) {
	// Go through the branches and ask the user for confirmation.
	var (
		localToDelete  = make([]string, 0, len(branches))
		remoteToDelete = make([]string, 0, len(branches))
	)

	defer fmt.Println()

	for _, branch := range branches {
		tip := branch.tip
		isLocal := tip.BranchName != ""
		isRemote := tip.RemoteBranchName != ""

		var msg string
		switch {
		case isLocal && isRemote:
			msg = fmt.Sprintf(
				"Processing local branch '%v' and its remote counterpart", tip.BranchName)
		case isLocal:
			msg = fmt.Sprintf(
				"Processing local branch '%v'", tip.BranchName)
		case isRemote:
			msg = fmt.Sprintf(
				"Processing remote branch '%v'", tip.FullRemoteBranchName())
		default:
			panic("bullshit")
		}
		fmt.Println()
		fmt.Println(msg)

		if branch.reason != "merged" {
			color.Yellow("Careful now, the branch has not been merged into trunk yet.")
		}

		confirmed, err := prompt.Confirm("Are you sure you want to delete the branch?", false)
		if err != nil {
			return nil, nil, err
		}
		if !confirmed {
			continue
		}

		if isLocal {
			localToDelete = append(localToDelete, tip.BranchName)
		}
		if isRemote {
			remoteToDelete = append(remoteToDelete, tip.RemoteBranchName)
		}
	}
	return localToDelete, remoteToDelete, nil
}
