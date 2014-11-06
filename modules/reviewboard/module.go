package reviewboard

import (
	// Stdlib
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strconv"

	// Internal
	"github.com/salsita/salsaflow/errs"
	"github.com/salsita/salsaflow/git"
	"github.com/salsita/salsaflow/log"
	"github.com/salsita/salsaflow/modules/common"
	"github.com/salsita/salsaflow/repo"
	"github.com/salsita/salsaflow/shell"
)

const Id = "review_board"

func init() {
	repo.AddInitHook(ensureRbtVersion)
}

type codeReviewTool struct{}

func Factory() (common.CodeReviewTool, error) {
	return &codeReviewTool{}, nil
}

func (tool *codeReviewTool) PostReviewRequest(commit *git.Commit, opts map[string]interface{}) error {
	// Assert that certain field are set.
	switch {
	case commit.SHA == "":
		panic("SHA not set for the commit being posted")
	case commit.StoryId == "":
		panic("story ID not set for the commit being posted")
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
	args := []string{"rbt", "post", "--guess-fields", "yes", "--bugs-closed", commit.StoryId}
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

	task := "Post review request for commit " + commit.SHA
	stdout, stderr, err := shell.Run(args...)
	if err != nil {
		return errs.NewError(task, err, stderr)
	}
	logger := log.V(log.Info)
	logger.Lock()
	logger.UnsafeNewLine("")
	logger.UnsafeOk(task)
	fmt.Print(stdout)
	logger.Unlock()
	return nil
}

func (tool *codeReviewTool) PrintPostReviewRequestFollowup() {
	log.Println(`
Now, take some time to go through all the review requests, please,
to check and annotate them for the reviewers to make their part easier.

If you find any issues you want to fix (even before publishing), do so now,
and if you haven't merged or pushed yet, amend the relevant commit and use

  $ salsaflow review post -update REVIEW_REQUEST_ID [REVISION]

to update (replace) the associated review request. Do this for every review
request you want to overwrite.

In case you cannot amend the relevant commits any more, make sure the affected
review request is published and use the usual process for fixing review issues

  $ salsaflow review post -fixes REVIEW_REQUEST_ID [REVISION]

to create a new review request that is linked to the one being fixed.
`)
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

	// rbt prints the version string to stderr. WHY? Who knows...
	_, stderr, err := shell.Run("rbt", "--version")
	if err != nil {
		return errs.NewError(task, err, stderr)
	}

	pattern := regexp.MustCompile("^RBTools (([0-9]+)[.]([0-9]+).*)")
	parts := pattern.FindStringSubmatch(stderr.String())
	if len(parts) != 4 {
		err := errors.New("failed to parse 'rbt --version' output: " + stderr.String())
		return errs.NewError(task, err, nil)
	}
	rbtVersion := parts[1]
	// No need to check errors, we know the format is correct.
	major, _ := strconv.Atoi(parts[2])
	minor, _ := strconv.Atoi(parts[3])

	if !(major == 0 && minor == 6) {
		hint := `
You need RBTools version 0.6. Please run

  $ pip install rbtools~=0.6 --allow-external rbtools --allow-unverified rbtools

to install the correct version.

`
		return errs.NewError(
			task,
			errors.New("unsupported rbt version detected: "+rbtVersion),
			bytes.NewBufferString(hint))
	}

	return nil
}
