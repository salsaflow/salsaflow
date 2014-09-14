package pruneCmd

import (
	// Stdlib
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"

	// Internal
	"github.com/tchap/git-trunk/app"
	"github.com/tchap/git-trunk/config"
	"github.com/tchap/git-trunk/git"
	"github.com/tchap/git-trunk/log"
	"github.com/tchap/git-trunk/prompt"
	pt "github.com/tchap/git-trunk/utils/pivotaltracker"

	// Other
	"github.com/tchap/gocli"
	"gopkg.in/salsita/go-pivotaltracker.v0/v5/pivotal"
)

var Command = &gocli.Command{
	UsageLine: `
  prune [-include_delivered] [-local_only] [-remote_only]`,
	Short: "prune delivered story branches",
	Long: `
  Delete local and remote branches that are associated with stories
  that were accepted (or delivered, that depends on the flags).
	`,
	Action: run,
}

var (
	flagIncludeDelivered bool
	flagLocalOnly        bool
	flagRemoteOnly       bool
)

func init() {
	Command.Flags.BoolVar(&flagIncludeDelivered, "include_delivered", flagIncludeDelivered,
		"prune Delivered story branches as well")
	Command.Flags.BoolVar(&flagLocalOnly, "local_only", flagLocalOnly,
		"prune local branches only")
	Command.Flags.BoolVar(&flagRemoteOnly, "remote_only", flagRemoteOnly,
		"prune remote branches only")
}

func run(cmd *gocli.Command, args []string) {
	if len(args) != 0 {
		cmd.Usage()
		os.Exit(2)
	}

	app.MustInit()

	if err := runMain(); err != nil {
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
		if err != nil {
			log.FailWithContext(msg, stderr)
		}
	}()

	// Fetch the remote repository unless we are restricted to the local branches only.
	if !flagLocalOnly {
		msg = "Fetch the remote repository"
		log.Run(msg)
		stderr, err = git.UpdateRemotes(config.OriginName)
		if err != nil {
			return
		}
	}

	// Get the list of story references.
	msg = "Collect all story branches"
	localRefs, remoteRefs, stderr, err := pt.ListGitStoryRefs()
	if err != nil {
		return
	}

	var refs []string
	switch {
	case flagLocalOnly:
		refs = localRefs
	case flagRemoteOnly:
		refs = remoteRefs
	default:
		refs = append(localRefs, remoteRefs...)
	}

	if len(refs) == 0 {
		msg = ""
		log.Println("\nNo relevant story branches found, exiting...")
		return
	}

	// Get the associated stories.
	msg = "Fetch the associated stories"
	log.Run(msg)
	idSet := make(map[int]struct{})
	for _, ref := range refs {
		id, _ := pt.RefToStoryId(ref)
		idSet[id] = struct{}{}
	}

	ids := make([]int, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}

	stories, stderr, err := pt.ListStoriesById(ids)
	if err != nil {
		return
	}

	storyMap := make(map[int]*pivotal.Story)
	for _, story := range stories {
		storyMap[story.Id] = story
	}

	// Filter the branches according to the story state.
	msg = "Filter the branches according to the story state"
	var filteredRefs []string
	for _, ref := range refs {
		id, _ := pt.RefToStoryId(ref)
		story, ok := storyMap[id]
		if !ok {
			err = fmt.Errorf("story with id %v not found", id)
			return
		}

		switch story.State {
		case pivotal.StoryStateAccepted:
			filteredRefs = append(filteredRefs, ref)
		case pivotal.StoryStateDelivered:
			if flagIncludeDelivered {
				filteredRefs = append(filteredRefs, ref)
			}
		}
	}
	refs = filteredRefs

	if len(refs) == 0 {
		msg = ""
		log.Println("\nThere are no branches to be deleted, exiting...")
		return
	}

	// Sort the refs.
	sort.Sort(sort.StringSlice(refs))

	// Prompt the user to confirm the delete operation.
	var (
		toDeleteLocally  []string
		toDeleteRemotely []string
		ok               bool
	)

	// Go through the local branches.
	if strings.HasPrefix(refs[0], "refs/heads/") {
		fmt.Println("\n---> Local branches\n")
	}
	for len(refs) > 0 {
		ref := refs[0]
		if !strings.HasPrefix(ref, "refs/heads/") {
			break
		}
		branch := ref[len("refs/heads/"):]
		question := fmt.Sprintf("Delete local branch '%v'", branch)
		ok, err = prompt.Confirm(question)
		if err != nil {
			return
		}
		if ok {
			toDeleteLocally = append(toDeleteLocally, branch)
		}
		refs = refs[1:]
	}

	// All that is left are remote branches.
	if len(refs) != 0 {
		fmt.Println("\n---> Remote branches\n")
	}
	for _, ref := range refs {
		branch := ref[len("refs/remotes/origin/"):]
		question := fmt.Sprintf("Delete remote branch '%v'", branch)
		ok, err = prompt.Confirm(question)
		if err != nil {
			return
		}
		if ok {
			toDeleteRemotely = append(toDeleteRemotely, branch)
		}
	}
	fmt.Println()

	if len(toDeleteLocally) == 0 && len(toDeleteRemotely) == 0 {
		msg = ""
		fmt.Println("No branches selected, exiting...")
		return
	}

	// Delete the local branches.
	if len(toDeleteLocally) != 0 {
		msg = "Delete the chosen local branches"
		log.Run(msg)

		// Remember the position of the branches to be deleted.
		// This is used in case we need to perform a rollback.
		var (
			currentPositions []string
			hexsha           string
		)
		for _, branchName := range toDeleteLocally {
			hexsha, stderr, err = git.Hexsha("refs/heads/" + branchName)
			if err != nil {
				return
			}
			currentPositions = append(currentPositions, hexsha)
		}

		// Delete the selected local branches.
		args := append([]string{"-d"}, toDeleteLocally...)
		stderr, err = git.Branch(args...)
		if err != nil {
			return
		}
		defer func(taskMsg string) {
			// On error, try to restore the local branches that were deleted.
			if err != nil {
				log.Rollback(taskMsg)
				for i, branchName := range toDeleteLocally {
					out, ex := git.ResetKeep(branchName, currentPositions[i])
					if ex != nil {
						log.FailWithContext(msg, out)
					}
				}
			}
		}(msg)
	}

	// Delete the remote branches.
	if len(toDeleteRemotely) != 0 {
		msg = "Delete the chosen remote branches"
		log.Run(msg)
		var refs []string
		for _, branchName := range toDeleteRemotely {
			refs = append(refs, ":"+branchName)
		}
		stderr, err = git.Push(config.OriginName, refs...)
	}
	return
}
