package reviewboard

import (
	// Stdlib
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strconv"

	// Internal
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/modules/common"
	"github.com/salsaflow/salsaflow/repo"
	"github.com/salsaflow/salsaflow/shell"
)

const Id = "review_board"

func init() {
	repo.AddInitHook(ensureRbtVersion)
}

type codeReviewTool struct{}

func Factory() (common.CodeReviewTool, error) {
	return &codeReviewTool{}, nil
}

func (tool *codeReviewTool) PostReviewRequestForCommit(
	ctx *common.CommitReviewContext,
	opts map[string]interface{},
) error {

	var (
		commit = ctx.Commit
		story  = ctx.Story
	)

	// Assert that certain field are set.
	switch {
	case commit.SHA == "":
		panic("SHA not set for the commit being posted")
	case commit.StoryIdTag == "":
		panic("story ID not set for the commit being posted")
	}

	// Load the RB config.
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	// Parse the options.
	var (
		fixes  = formatOptInteger(opts["fixes"])
		update = formatOptInteger(opts["update"])
		open   bool
	)
	if _, ok := opts["open"]; ok {
		open = true
	}

	// Post the review request.
	args := []string{"post",
		"--server", config.ServerURL().String(),
		"--guess-fields", "yes",
	}

	if story != nil {
		args = append(args, "--bugs-closed", commit.StoryIdTag)
	}
	if fixes != "" {
		args = append(args, "--depends-on", fixes)
	}
	if update != "" {
		args = append(args, "--review-request-id", update)
	}
	if open {
		args = append(args, "--open")
	}
	args = append(args, commit.SHA)

	var task string
	if update != "" {
		task = "Update a Review Board review request with commit " + commit.SHA
	} else {
		task = "Create a Review Board review request for commit " + commit.SHA
	}
	log.Run(task)
	stdout, stderr, err := shell.Run("rbt", args...)
	if err != nil {
		// rbt is retarded and sometimes prints stderr to stdout.
		// That is why we return stdout when stderr is empty.
		if stderr.Len() == 0 {
			return errs.NewError(task, err, stdout)
		} else {
			return errs.NewError(task, err, stderr)
		}
	}
	logger := log.V(log.Info)
	logger.Lock()
	logger.UnsafeNewLine("")
	logger.UnsafeOk(task)
	fmt.Print(stdout)
	logger.Unlock()
	return nil
}

func (tool *codeReviewTool) PostReviewRequestForBranch(
	branch string,
	ctxs []*common.CommitReviewContext,
	opts map[string]interface{},
) (err error) {

	// Use PostReviewRequestForCommit for every commit on the branch.
	// Try to post a review request for every commit and keep printing the errors.
	// Return a common error in case there is any partial error encountered.
	for _, ctx := range ctxs {
		if ex := tool.PostReviewRequestForCommit(ctx, opts); ex != nil {
			log.NewLine("")
			errs.Log(ex)
			err = errors.New("failed to post a review request")
		}
	}
	return
}

func (tool *codeReviewTool) PostReviewFollowupMessage() string {
	return `
Now, please, take some time to go through all the review requests
to check and annotate them for the reviewers to make their part easier.

If you find any issues you want to fix (even before publishing),
do so now, and if you haven't pushed into any shared branch yet,
amend the relevant commit and use

  $ salsaflow review post -update REVIEW_REQUEST_ID [REVISION]

to update (replace) the associated review request. Do this for every review
request you want to overwrite.

In case you cannot amend the relevant commit any more, make sure the affected
review request is published, and use the process for fixing review issues:

  $ salsaflow review post -fixes REVIEW_REQUEST_ID [REVISION]

This will create a new review request that is linked to the one being fixed.

  ########################################################################
  # IMPORTANT: Your code has not been merged and/or pushed anywhere yet. #
  ########################################################################
`
}

func formatOptInteger(value interface{}) string {
	// Return an empty string on nil.
	if value == nil {
		return ""
	}

	// Return an empty string in case the value is not an integer.
	switch value := value.(type) {
	case string:
		return value
	case int:
	case int32:
	case int64:
	case uint:
	case uint32:
	case uint64:
	default:
		return ""
	}

	// Format the integer and return the string representation.
	return fmt.Sprintf("%v", value)
}

func ensureRbtVersion() error {
	hint := bytes.NewBufferString(`
You need to install RBTools version 0.7. Please run

  $ pip install rbtools==0.7 --allow-external rbtools --allow-unverified rbtools

to install the correct version.

`)
	// Load configuration and check the RBTools version only if Review Board is being used.
	config, err := common.LoadConfig()
	if err != nil {
		return err
	}
	if config.CodeReviewToolId() != Id {
		return nil
	}

	// Check the RBTools version being used.
	task := "Check the RBTools version being used"
	log.Run(task)

	// rbt 0.5.x prints the version string to stdout,
	// rbt 0.6.x prints the version string to stderr.
	stdout, stderr, err := shell.Run("rbt", "--version")
	if err != nil {
		// Return the hint instead of stderr.
		// Failing to run rbt --version probably means that it's not installed.
		return errs.NewError(task, err, hint)
	}

	var outputBuffer *bytes.Buffer
	if stdout.Len() != 0 {
		outputBuffer = stdout
	} else {
		outputBuffer = stderr
	}
	output := outputBuffer.String()

	pattern := regexp.MustCompile("^RBTools (([0-9]+)[.]([0-9]+).*)")
	parts := pattern.FindStringSubmatch(output)
	if len(parts) != 4 {
		err := fmt.Errorf("failed to parse 'rbt --version' output: %v", output)
		return errs.NewError(task, err, nil)
	}
	rbtVersion := parts[1]
	// No need to check errors, we know the format is correct.
	major, _ := strconv.Atoi(parts[2])
	minor, _ := strconv.Atoi(parts[3])

	if !(major == 0 && minor == 7) {
		return errs.NewError(
			task, errors.New("unsupported rbt version detected: "+rbtVersion), hint)
	}

	return nil
}
