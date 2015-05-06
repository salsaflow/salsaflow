package main

import (
	// Stdlib
	"fmt"
	"os"
	"regexp"
	"strings"

	// Internal
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/git"
	"github.com/salsaflow/salsaflow/hooks"

	// Vendor
	"github.com/fatih/color"
)

func main() {
	// Register the magical -salsaflow.version flag.
	hooks.IdentifyYourself()

	// Run the hook logic itself.
	if err := hook(); err != nil {
		errs.Fatal(err)
	}
}

func hook() error {
	// There are always 3 arguments passed to this hook.
	prevRef, newRef, flag := os.Args[1], os.Args[2], os.Args[3]

	// Return in case prevRef is the zero hash since that means
	// that this hook is being run right after 'git clone'.
	if prevRef == git.ZeroHash {
		return nil
	}

	// Return in case flag is '0'. That signals retrieving a file from the index.
	if flag == "0" {
		return nil
	}

	// Return unless the new HEAD is a core branch.
	isCore, err := isCoreBranch(newRef)
	if err != nil {
		return err
	}
	if !isCore {
		return nil
	}

	// Get the relevant commits.
	// These are the commits specified by newRef..prevRef, e.g. trunk..story/foobar.
	commits, err := git.ShowCommitRange(fmt.Sprintf("%v..%v", newRef, prevRef))
	if err != nil {
		return err
	}

	// Collect the commits with missing Story-Id tag.
	missing := make([]*git.Commit, 0, len(commits))
	for _, commit := range commits {
		if commit.StoryIdTag == "" {
			missing = append(missing, commit)
		}
	}
	if len(missing) == 0 {
		return nil
	}

	// Print the warning.
	printWarning(missing)
	return nil
}

func isCoreBranch(ref string) (bool, error) {
	// Get the ref names using 'git log'.
	// This prints the ref names in the following format:
	//
	//   (ref1, ref2, ..., refN)
	//
	outputBuffer, err := git.Log("-1", "--pretty=format:%d", ref)
	if err != nil {
		return false, err
	}
	output := outputBuffer.String()

	// Parse the output.
	match := regexp.MustCompile("^[ ]*[(]([^)]+)[)][ ]*$").FindStringSubmatch(output)
	if len(match) != 2 {
		return false, fmt.Errorf("failed to parse git log: %v", output)
	}
	refNames := strings.Split(match[1], ", ")

	// Iterate over the ref names and return the result.
	for _, ref := range refNames {
		isCore, err := git.IsCoreBranch(ref)
		if err != nil {
			return false, err
		}
		if isCore {
			return true, nil
		}
	}
	return false, nil
}

func printWarning(commits []*git.Commit) {
	// Let's be colorful!
	redBold := color.New(color.FgRed).Add(color.Bold)
	redBold.Println("\nWarning: There are some commits missing the Story-Id tag.")

	red := color.New(color.FgRed)
	red.Println("Make sure this is really what you want before proceeding further.\n")

	yellow := color.New(color.FgYellow).SprintFunc()
	for _, commit := range commits {
		fmt.Printf("  %v %v\n", yellow(commit.SHA), commit.MessageTitle)
	}
	fmt.Println()
}
