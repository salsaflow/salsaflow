package git

import (
	// Stdlib
	"bufio"
	"fmt"
	"regexp"
	"strings"

	// Internal
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/log"
)

// Branch represents a Git branch.
type GitBranch struct {
	// Local branch name
	BranchName string

	// Name of the remote branch being tracked
	RemoteBranchName string

	// Name of remote the remote branch is associated with
	Remote string
}

// LocalRef returns "refs/heads/{{.BranchName}}".
func (branch *GitBranch) LocalRef() string {
	if branch.BranchName == "" {
		return ""
	}
	return fmt.Sprintf("refs/heads/%v", branch.BranchName)
}

// RemoteRef returns "refs/remotes/{{.Remote}}/{{.RemoteBranchName}}".
func (branch *GitBranch) RemoteRef() string {
	if branch.RemoteBranchName == "" {
		return ""
	}
	return fmt.Sprintf("refs/remotes/%v/%v", branch.Remote, branch.RemoteBranchName)
}

// FullRemoteName returns "{{.Remote}}/{{.RemoteBranchName}}".
func (branch *GitBranch) FullRemoteBranchName() string {
	return fmt.Sprintf("%v/%v", branch.Remote, branch.RemoteBranchName)
}

// IsUpToDate returns true when the local and remote references
// point to the same commit.
func (branch *GitBranch) IsUpToDate() (bool, error) {
	var (
		localRef  = branch.LocalRef()
		remoteRef = branch.RemoteRef()
	)

	// Return true in case this is a purely local or remote branch.
	if len(localRef) == 0 || len(remoteRef) == 0 {
		return true, nil
	}

	// Compare the hashes the branches are pointing to.
	localHexsha, err := Hexsha(branch.LocalRef())
	if err != nil {
		return false, err
	}
	remoteHexsha, err := Hexsha(branch.RemoteRef())
	if err != nil {
		return false, err
	}
	return localHexsha == remoteHexsha, nil
}

func Branches() ([]*GitBranch, error) {
	// Get local branches.
	local, err := localBranches()
	if err != nil {
		return nil, err
	}

	// Get remote branches.
	remote, err := remoteBranches()
	if err != nil {
		return nil, err
	}

	// Clean up the local branches.
	// In can happen that the tracked branch fields are set while the branch
	// itself doesn't exist any more since the git calls are only consulting
	// .git/config. They don't really care whether the branch actually exists.
LocalLoop:
	for _, localBranch := range local {
		// In case the remote record is empty, we are obviously cool.
		if localBranch.RemoteBranchName == "" {
			continue
		}

		// Otherwise go through the remote branches and only continue
		// when the corresponding remote branch is found.
		for _, remoteBranch := range remote {
			if remoteBranch.RemoteBranchName == localBranch.RemoteBranchName {
				continue LocalLoop
			}
		}

		// In case the remote branch is missing, clean up the record in .git/config.
		branchName := localBranch.BranchName
		log.Warn(fmt.Sprintf(
			"Branch '%v/%v' not found", localBranch.FullRemoteBranchName()))
		log.NewLine(fmt.Sprintf("Unsetting upstream for local branch '%v'", branchName))

		task := fmt.Sprintf("Unset upstream branch for branch '%v'", branchName)
		if err := Branch("--unset-upstream", branchName); err != nil {
			return nil, errs.NewError(task, err)
		}

		// Unset the remote branch fields.
		localBranch.RemoteBranchName = ""
		localBranch.Remote = ""
	}

	// Append the remote branch records to the local ones.
	// Only include these that are not already included in the local records.
	branches := local
RemoteLoop:
	for _, remoteBranch := range remote {
		for _, localBranch := range local {
			if localBranch.RemoteBranchName == remoteBranch.RemoteBranchName {
				continue RemoteLoop
			}
		}
		branches = append(branches, remoteBranch)
	}

	// Return branches.
	return branches, nil
}

func localBranches() ([]*GitBranch, error) {
	// Get raw data.
	task := "Get Git branch data"
	stdout, err := Run("branch", "-vv")
	if err != nil {
		return nil, errs.NewError(task, err)
	}

	// Parse the data.
	task = "Parse 'git branch -vv' output"
	scanner := bufio.NewScanner(stdout)
	lineRegexp := regexp.MustCompile(`^[ *]+([^ \t]+)[ \t]+[^ \t]+[ \t]+(\[([^\]:]+))?`)
	branches := make([]*GitBranch, 0)

	for scanner.Scan() {
		line := scanner.Text()
		match := lineRegexp.FindStringSubmatch(line)
		if len(match) == 0 {
			err := fmt.Errorf("failed to parse output line: %v", line)
			return nil, errs.NewError(task, err)
		}

		branch := match[1]
		var remote, remoteBranch string

		parts := strings.SplitN(match[3], "/", 2)
		if len(parts) == 2 {
			remote, remoteBranch = parts[0], parts[1]
		}

		branches = append(branches, &GitBranch{
			BranchName:       branch,
			RemoteBranchName: remoteBranch,
			Remote:           remote,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, errs.NewError(task, err)
	}
	return branches, nil
}

func remoteBranches() ([]*GitBranch, error) {
	// Get raw data.
	task := "Get Git branch data"
	stdout, err := Run("branch", "-rvv")
	if err != nil {
		return nil, errs.NewError(task, err)
	}

	// Parse the data.
	task = "Parse 'git branch -rvv' output"
	scanner := bufio.NewScanner(stdout)
	lineRegexp := regexp.MustCompile(`^[ *]+([^ \t]+)`)
	branches := make([]*GitBranch, 0)

	for scanner.Scan() {
		line := scanner.Text()
		match := lineRegexp.FindStringSubmatch(line)
		if len(match) == 0 {
			err := fmt.Errorf("failed to parse output line: %v", line)
			return nil, errs.NewError(task, err)
		}

		var remote, remoteBranch string
		parts := strings.SplitN(match[1], "/", 2)
		remote, remoteBranch = parts[0], parts[1]

		// Do not return HEAD.
		if remoteBranch == "HEAD" {
			continue
		}

		branches = append(branches, &GitBranch{
			RemoteBranchName: remoteBranch,
			Remote:           remote,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, errs.NewError(task, err)
	}
	return branches, nil
}
