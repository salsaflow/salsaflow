package reviewboard

import (
	// Stdlib
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	// Internal
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/metastore"
	"github.com/salsaflow/salsaflow/modules/common"
	"github.com/salsaflow/salsaflow/repo"
	"github.com/salsaflow/salsaflow/shell"
	"github.com/salsaflow/salsaflow/version"
)

const Id = "review_board"

func init() {
	// Load common configuration.
	config, err := common.LoadConfig()
	if err != nil {
		// Just do nothing on error.
		return
	}

	// Register repo init hook in case we are using RB.
	if config.CodeReviewToolId() == Id {
		repo.AddInitHook(ensureRbtVersion)
	}
}

type codeReviewTool struct{}

func Factory() (common.CodeReviewTool, error) {
	return &codeReviewTool{}, nil
}

func (tool *codeReviewTool) InitialiseRelease(v *version.Version) (action.Action, error) {
	return action.ActionFunc(func() error { return nil }), nil
}

func (tool *codeReviewTool) FinaliseRelease(v *version.Version) (action.Action, error) {
	return action.ActionFunc(func() error { return nil }), nil
}

func (tool *codeReviewTool) PostReviewRequests(
	ctxs []*common.ReviewContext,
	opts map[string]interface{},
) ([]*common.ReviewContext, error) {

	// Use postReviewRequestForCommit for every commit on the branch.
	// Try to post a review request for every commit and keep printing the errors.
	// Return a common error in case there is any partial error encountered.
	var (
		newCtxs = make([]*common.ReviewContext, 0, len(ctxs))
		err     error
	)
	for _, ctx := range ctxs {
		newCtx, ex := postReviewRequestForCommit(ctx, opts)
		if ex != nil {
			log.NewLine("")
			errs.Log(ex)
			err = errors.New("failed to post a review request")
		}
		newCtxs = append(newCtxs, newCtx)
	}
	if err != nil {
		return nil, err
	}
	return newCtxs, nil
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
`
}

func postReviewRequestForCommit(
	ctx *common.ReviewContext,
	opts map[string]interface{},
) (*common.ReviewContext, error) {

	var (
		commit = ctx.Commit
		story  = ctx.Story
	)

	// Load the RB config.
	config, err := LoadConfig()
	if err != nil {
		return nil, err
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
		args = append(args, "--bugs-closed", story.Tag())
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

	var (
		task   string
		newCtx *common.ReviewContext
	)
	if update != "" {
		task = "Update a Review Board review request with commit " + commit.SHA
		newCtx = ctx
	} else {
		task = "Create a Review Board review request for commit " + commit.SHA
	}
	log.Run(task)
	stdout, stderr, err := shell.Run("rbt", args...)
	if err != nil {
		// rbt is retarded and sometimes prints stderr to stdout.
		// That is why we return stdout when stderr is empty.
		if stderr.Len() == 0 {
			return nil, errs.NewErrorWithHint(task, err, stdout.String())
		} else {
			return nil, errs.NewErrorWithHint(task, err, stderr.String())
		}
	}

	out := stdout.String()
	logger := log.V(log.Info)
	logger.Lock()
	logger.UnsafeNewLine("")
	logger.UnsafeOk(task)
	fmt.Print(out)
	logger.Unlock()

	if update == "" {
		resource, err := parseRbtOutput(out)
		if err != nil {
			return nil, err
		}
		newCtx = &common.ReviewContext{
			Commit:        ctx.Commit,
			Story:         ctx.Story,
			ReviewRequest: resource,
		}
	}
	return newCtx, nil
}

func parseRbtOutput(out string) (*metastore.Resource, error) {
	// The format is:
	//
	// Review request #12832 posted.
	//
	// https://review.example.com/r/12832/
	// https://review.example.com/r/12832/diff/

	task := "Parse rbt output"
	lines := strings.Split(out, "\n")

	match := regexp.MustCompile("^Review request #([0-9]+) posted[.]$").FindStringSubmatch(lines[0])
	if len(match) != 2 {
		hint := fmt.Sprintf("failed to parse line 1: %v", lines[0])
		return nil, errs.NewErrorWithHint(task, errors.New("failed to parse rbt output"), hint)
	}
	rrid, _ := strconv.Atoi(match[1])
	link := lines[2]

	return &metastore.Resource{
		ServiceId: "reviewboard",
		Metadata: map[string]interface{}{
			"rr_id":   rrid,
			"rr_link": link,
		},
	}, nil
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
	hint := `
You need to install RBTools version 0.7. Please run

  $ pip install rbtools==0.7 --allow-external rbtools --allow-unverified rbtools

to install the correct version.

`

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
		return errs.NewErrorWithHint(task, err, hint)
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
		return errs.NewError(task, err)
	}
	rbtVersion := parts[1]
	// No need to check errors, we know the format is correct.
	major, _ := strconv.Atoi(parts[2])
	minor, _ := strconv.Atoi(parts[3])

	if !(major == 0 && minor == 7) {
		return errs.NewErrorWithHint(
			task, errors.New("unsupported rbt version detected: "+rbtVersion), hint)
	}

	return nil
}
