package pruneCmd

import (
	// Stdlib
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"

	// Internal
	"github.com/salsaflow/salsaflow/app"
	"github.com/salsaflow/salsaflow/config"
	"github.com/salsaflow/salsaflow/git"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/modules"
	"github.com/salsaflow/salsaflow/prompt"

	// Other
	"gopkg.in/tchap/gocli.v2"
)

var Command = &gocli.Command{
	UsageLine: `
  prune [-local_only] [-remote_only]`,
	Short: "prune delivered story branches",
	Long: `
  Delete local and remote branches that are associated with stories
  that are potentially finished are can be pruned.

  Only the branches owned by the caller are offered for deletion.
	`,
	Action: run,
}

var (
	flagIncludeDelivered bool
	flagAllOwners        bool
	flagLocalOnly        bool
	flagRemoteOnly       bool
)

func init() {
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
		task   string
		stderr *bytes.Buffer
	)
	defer func() {
		// Print error details.
		if err != nil {
			log.FailWithDetails(task, stderr)
		}
	}()

	// Fetch the remote repository unless we are restricted to the local branches only.
	if !flagLocalOnly {
		task = "Fetch the remote repository"
		log.Run(task)
		stderr, err = git.UpdateRemotes(config.OriginName)
		if err != nil {
			return
		}
	}

	// Get the list of story references.
	task = "Collect all story branches"
	log.Run(task)
	localRefs, remoteRefs, stderr, err := git.ListStoryRefs()
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
		task = ""
		log.Println("\nNo story branches found, exiting...")
		return
	}

	// Collect all the story IDs.
	idMap := make(map[string]struct{})
	for _, ref := range refs {
		// This cannot fail here since we got the refs using ListStoryRefs.
		id, _ := git.RefToStoryId(ref)
		idMap[id] = struct{}{}
	}

	var ids []string
	for id := range idMap {
		ids = append(ids, id)
	}

	// Get the list of active story IDs.
	activeIds, err := modules.GetIssueTracker().SelectActiveStoryIds(ids)
	if err != nil {
		return
	}
	ids = activeIds

	// Select only the refs that can be safely deleted.
	refs = selectInactiveRefs(refs, ids)

	if len(refs) == 0 {
		task = ""
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
		task = ""
		fmt.Println("No branches selected, exiting...")
		return
	}

	// Delete the local branches.
	if len(toDeleteLocally) != 0 {
		task = "Delete the chosen local branches"
		log.Run(task)

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
						log.FailWithDetails(task, out)
					}
				}
			}
		}(task)
	}

	// Delete the remote branches.
	if len(toDeleteRemotely) != 0 {
		task = "Delete the chosen remote branches"
		log.Run(task)
		var refs []string
		for _, branchName := range toDeleteRemotely {
			refs = append(refs, ":"+branchName)
		}
		stderr, err = git.Push(config.OriginName, refs...)
	}
	return
}

// selectInactiveRefs returns the list of refs that can be safely deleted.
// A reference can be safely deleted when the associated story is closed.
func selectInactiveRefs(refs, activeIds []string) (inactiveRefs []string) {
	var suffixes []string
	for _, id := range activeIds {
		suffixes = append(suffixes, "/"+id)
	}

	var inactive []string
RefLoop:
	for _, ref := range refs {
		for _, suffix := range suffixes {
			if strings.HasSuffix(ref, suffix) {
				continue RefLoop
			}
		}
		inactive = append(inactive, ref)
	}

	return inactive
}
