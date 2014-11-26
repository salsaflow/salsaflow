package deployCmd

import (
	// Stdlib
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	// Internal
	"github.com/salsaflow/salsaflow/app"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/git"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/modules"
	"github.com/salsaflow/salsaflow/modules/common"
	"github.com/salsaflow/salsaflow/prompt"
	"github.com/salsaflow/salsaflow/releases"
	"github.com/salsaflow/salsaflow/version"

	// Other
	"gopkg.in/tchap/gocli.v1"
)

var Command = &gocli.Command{
	UsageLine: "deploy [-release=RELEASE_TAG]",
	Short:     "deploy a release into production",
	Long: `
  Deploy the chosen release into production.

  This basically means that the stable branch is reset
  to point to the relevant release tag, then force pushed.

  In case the release is not specified explicitly, the user is offered
  the releases that can be deployed. These are the releases that happened
  after the current stable branch position. On top of that,
  all associated stories must be accepted.

  In case the release is specified on the command line, no additional checks
  are performed and the stable branch is reset and pushed. USE WITH CAUTION!
	`,
	Action: run,
}

var flagRelease string

func init() {
	Command.Flags.StringVar(&flagRelease, "release", flagRelease, "release tag to deploy")
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
	// Load repo config.
	gitConfig, err := git.LoadConfig()
	if err != nil {
		return err
	}

	var (
		remoteName   = gitConfig.RemoteName()
		stableBranch = gitConfig.StableBranchName()
	)

	// Make sure the stable branch exists.
	task := fmt.Sprintf("Make sure branch '%v' exists", stableBranch)
	if err := git.CreateTrackingBranchUnlessExists(stableBranch, remoteName); err != nil {
		return errs.NewError(task, err, nil)
	}

	// Make sure we are not on the stable branch.
	task = fmt.Sprintf("Make sure we are not on branch '%v'", stableBranch)
	currentBranch, err := git.CurrentBranch()
	if err != nil {
		return errs.NewError(task, err, nil)
	}

	if currentBranch == stableBranch {
		err := fmt.Errorf("cannot deploy while on branch '%v'", stableBranch)
		return errs.NewError(task, err, nil)
	}

	// In case the release is specified explicitly, just do the reset and return.
	if ref := flagRelease; ref != "" {
		if err := ensureRefExists(ref); err != nil {
			return errs.NewError(task, err, nil)
		}
		return resetAndDeploy(stableBranch, flagRelease, remoteName)
	}

	// Get the list of release tags since the last deployment.
	task = "Get the list of deployable releases"
	tags, err := newReleaseTags(stableBranch)
	if err != nil {
		return errs.NewError(task, err, nil)
	}

	// Limit the list to the releases that are fully accepted.
	tracker, err := modules.GetIssueTracker()
	if err != nil {
		return errs.NewError(task, err, nil)
	}

	var releasable []common.RunningRelease
	for _, tag := range tags {
		ver, err := version.FromTag(tag)
		if err != nil {
			return err
		}

		release, err := tracker.RunningRelease(ver)
		if err != nil {
			if errs.RootCause(err) == common.ErrReleaseNotFound {
				log.Warn(fmt.Sprintf("Release '%v' not found in the issue tracker", tag))
				continue
			}
			return errs.NewError(task, err, nil)
		}

		stories, err := release.CheckReleasable()
		if err != nil {
			return errs.NewError(task, err, nil)
		}
		if len(stories) != 0 {
			log.Log(fmt.Sprintf("Release '%v' is not releasable", tag))
			continue
		}

		releasable = append(releasable, release)
	}
	if len(releasable) == 0 {
		return errs.NewError(task, errors.New("no deployable releases found"), nil)
	}

	// Prompt the user to choose the release tag.
	task = "Prompt the user to choose the release to be deployed"
	fmt.Printf("\nThe following releases can be deployed:\n\n")
	tw := tabwriter.NewWriter(os.Stdout, 0, 8, 4, '\t', 0)
	io.WriteString(tw, "Index\tRelease\n")
	io.WriteString(tw, "=====\t=======\n")
	for i, release := range releasable {
		fmt.Fprintf(tw, "%v\t%v\n", i, release.Version())
	}
	tw.Flush()
	fmt.Println()

	index, err := prompt.PromptIndex(
		"Choose the release to be deployed by entering its index: ", 0, len(tags)-1)
	if err != nil {
		if err == prompt.ErrCanceled {
			prompt.PanicCancel()
		}
		return errs.NewError(task, err, nil)
	}
	fmt.Println()

	// Reset and push the stable branch.
	targetTag := releasable[index].Version().ReleaseTagString()
	if err := resetAndDeploy(stableBranch, targetTag, remoteName); err != nil {
		return err
	}

	// Release all the affected releases, one by one.
	//
	// There usually won't be that many releases, so let's skip concurrency.
	//
	// In case there is an error, tell the details to the user and let them
	// handle the cleanup since it's not possible to easily rollback the push.
	for _, release := range releasable[:index+1] {
		releaseName := release.Version().ReleaseTagString()
		task := fmt.Sprintf("Mark release '%v' as released", releaseName)
		log.Run(task)
		err = release.Release()
		if err != nil {
			err = errs.Log(errs.NewError(task, err, nil))
			continue
		}
	}
	if err != nil {
		logger := log.V(log.Info)
		logger.Lock()
		log.UnsafeWarn("Errors encountered while releasing stories in the issue tracker")
		log.UnsafeNewLine("Please perform the release in the issue tracker manually")
		log.UnsafeNewLine("to make sure the issue tracker is consistent.")
		logger.Unlock()
	}
	return err
}

func ensureRefExists(ref string) error {
	task := fmt.Sprintf("Make sure ref '%v' exists", ref)
	exists, err := git.RefExists(ref)
	if err != nil {
		return errs.NewError(task, err, nil)
	}
	if !exists {
		return errs.NewError(task, fmt.Errorf("ref '%v' not found", ref), nil)
	}
	return nil
}

func resetAndDeploy(stableBranch, targetRef, remoteName string) error {
	// Get the current stable branch position.
	task := fmt.Sprintf("Remember the current for branch '%v'", stableBranch)
	originalPosition, err := git.Hexsha("refs/heads/" + stableBranch)
	if err != nil {
		return errs.NewError(task, err, nil)
	}

	// Reset the stable branch to point to the target ref.
	resetTask := fmt.Sprintf("Reset branch '%v' to point to '%v'", stableBranch, targetRef)
	log.Run(resetTask)
	if err := git.Branch("-f", stableBranch, targetRef); err != nil {
		return errs.NewError(task, err, nil)
	}

	// Push the stable branch to deploy.
	task = fmt.Sprintf("Push branch '%v' to remote '%v'", stableBranch, remoteName)
	log.Run(task)
	err = git.PushForce(remoteName, fmt.Sprintf("%v:%v", stableBranch, stableBranch))
	if err != nil {
		// On error, reset the stable branch to the original position.
		log.Rollback(resetTask)
		if ex := git.Branch("-f", stableBranch, originalPosition); ex != nil {
			errs.LogError(
				fmt.Sprintf("Reset branch '%v' to the original position", stableBranch), ex, nil)
		}
		return errs.NewError(task, err, nil)
	}

	return nil
}

func newReleaseTags(stableBranch string) ([]string, error) {
	// Get the list of all release tags.
	tags, err := releases.ListTags()
	if err != nil {
		return nil, err
	}
	if len(tags) == 0 {
		return nil, nil
	}

	// Get the tag pointing to the stable branch.
	//
	// Here we count on the fact that the stable branch is always tagged
	// when release deploy is being called since release stage must have been called before.
	// This is the simplest way to go around various git pains.
	task := fmt.Sprintf("Get the tag pointing to the tip of branch '%v'", stableBranch)
	stdout, err := git.Run("describe", "--tags", "--exact-match", stableBranch)
	if err != nil {
		return nil, errs.NewError(task, err, nil)
	}
	deployedTag := strings.TrimSpace(stdout.String())

	// Get the new tags.
	//
	// Keep dropping tags until we encounter the deployed tag.
	// Since the tags are sorted, the remaining tags are the new tags.
	var offset int
	for _, tag := range tags {
		if tag == deployedTag {
			break
		}
		offset++
	}
	tags = tags[offset+1:]
	return tags, nil
}
