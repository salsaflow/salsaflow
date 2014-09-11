package pruneCmd

import (
	// Stdlib
	"bytes"
	"errors"
	"fmt"
	"os"

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

// XXX: THIS IS TERRIBLY UGLY, REWRITE!
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
	local, remote, stderr, err := pt.ListStoryRefs()
	if err != nil {
		return
	}

	if flagLocalOnly {
		remote = nil
	}
	if flagRemoteOnly {
		local = nil
	}

	if len(local) == 0 && len(remote) == 0 {
		msg = ""
		log.Println("\nNo relevant story branches found, exiting...")
		return
	}

	// Get the associated stories.
	msg = "Fetch the associated stories"
	log.Run(msg)
	idSet := make(map[int]struct{})
	for _, ref := range local {
		id, _ := pt.RefToStoryId(ref)
		idSet[id] = struct{}{}
	}
	for _, ref := range remote {
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
	var (
		newLocal  []string
		newRemote []string
	)
	for _, ref := range local {
		id, _ := pt.RefToStoryId(ref)
		story, ok := storyMap[id]
		if !ok {
			err = fmt.Errorf("story with id=%v not found", id)
			return
		}

		switch story.State {
		case pivotal.StoryStateAccepted:
			newLocal = append(newLocal, ref)
		case pivotal.StoryStateDelivered:
			if flagIncludeDelivered {
				newLocal = append(newLocal, ref)
			}
		}
	}
	local = newLocal

	for _, ref := range remote {
		id, _ := pt.RefToStoryId(ref)
		story, ok := storyMap[id]
		if !ok {
			err = fmt.Errorf("story with id=%v not found", id)
			return
		}

		switch story.State {
		case pivotal.StoryStateAccepted:
			newRemote = append(newRemote, ref)
		case pivotal.StoryStateDelivered:
			if flagIncludeDelivered {
				newRemote = append(newRemote, ref)
			}
		}
	}
	remote = newRemote

	if len(local) == 0 && len(remote) == 0 {
		msg = ""
		log.Println("\nThere are no branches to be deleted, exiting...")
		return
	}

	// Prompt the user to confirm the branches.
	var (
		toDeleteLocally  []string
		toDeleteRemotely []string
		ok               bool
	)
	if len(local) != 0 {
		fmt.Println("\n---> Local branches\n")
	}
	for _, ref := range local {
		branch := ref[len("refs/heads/"):]
		q := fmt.Sprintf("Delete local branch '%v'?", ref[len("refs/heads/"):])
		ok, err = prompt.Confirm(q)
		if err != nil {
			return
		}
		if ok {
			toDeleteLocally = append(toDeleteLocally, branch)
		}
	}
	if len(remote) != 0 {
		fmt.Println("\n---> Remote branches\n")
	}
	for _, ref := range remote {
		branch := ref[len("refs/remotes/"):]
		q := fmt.Sprintf("Delete remote branch '%v'?", branch)
		ok, err = prompt.Confirm(q)
		if err != nil {
			return
		}
		if ok {
			toDeleteRemotely = append(toDeleteRemotely, branch[len(config.OriginName)+1:])
		}
	}
	fmt.Println()

	if len(toDeleteLocally) == 0 && len(toDeleteRemotely) == 0 {
		err = errors.New("No branches selected, operation canceled...")
		return
	}

	// Delete the local branches.
	if len(toDeleteLocally) != 0 {
		msg = "Delete the chosen local branches"
		log.Run(msg)

		// Remember the position of the branches to be deleted.
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
