package git

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

var ErrDirtyRepository = errors.New("the repository is dirty")

func UpdateRemotes(remotes ...string) (stderr *bytes.Buffer, err error) {
	argsList := append([]string{"remote", "update"}, remotes...)
	_, stderr, err = Git(argsList...)
	return
}

func Push(remote string, args ...string) (stderr *bytes.Buffer, err error) {
	argsList := append([]string{"push"}, remote)
	argsList = append(argsList, args...)
	_, stderr, err = Git(argsList...)
	return
}

func Branch(args ...string) (stderr *bytes.Buffer, err error) {
	argsList := append([]string{"branch"}, args...)
	_, stderr, err = Git(argsList...)
	return
}

func RefExists(ref string) (exists bool, stderr *bytes.Buffer, err error) {
	_, out, err := Git("show-ref", "--quiet", ref)
	if err != nil {
		if out.Len() != 0 {
			// Non-empty error output means that there was an error.
			return false, out, err
		}
		// Otherwise the ref does not exist.
		return false, out, nil
	}
	// No error means that the ref exists.
	return true, out, nil
}

// RefExistsStrict requires the whole ref path to be specified,
// e.g. refs/remotes/origin/master.
func RefExistsStrict(ref string) (exists bool, stderr *bytes.Buffer, err error) {
	_, out, err := Git("show-ref", "--verify", "--quiet", ref)
	if err != nil {
		if out.Len() != 0 {
			// Non-empty error output means that there was an error.
			return false, out, err
		}
		// Otherwise the ref does not exist.
		return false, out, nil
	}
	// No error means that the ref exists.
	return true, out, nil
}

func EnsureBranchNotExists(branch string, remote string) (stderr *bytes.Buffer, err error) {
	exists, stderr, err := LocalBranchExists(branch)
	if err != nil {
		return
	}
	if exists {
		err = fmt.Errorf("branch '%v' already exists", branch)
		return
	}

	exists, stderr, err = RemoteBranchExists(branch, remote)
	if err != nil {
		return
	}
	if exists {
		err = fmt.Errorf("branch '%v' already exists in remote '%v'", branch, remote)
	}
	return
}

func LocalBranchExists(branch string) (exists bool, stderr *bytes.Buffer, err error) {
	ref := "refs/heads/" + branch
	return RefExistsStrict(ref)
}

func RemoteBranchExists(branch string, remote string) (exists bool, stderr *bytes.Buffer, err error) {
	ref := fmt.Sprintf("refs/remotes/%v/%v", remote, branch)
	return RefExistsStrict(ref)
}

func CreateTrackingBranchUnlessExists(branch string, remote string) (stderr *bytes.Buffer, err error) {
	// Check whether the local branch exists and just return in that case.
	exists, stderr, err := LocalBranchExists(branch)
	if exists || err != nil {
		return
	}

	// Check whether the remote counterpart exists.
	exists, stderr, err = RemoteBranchExists(branch, remote)
	if err != nil {
		return
	}
	if !exists {
		err = fmt.Errorf("branch '%v' not found in the remote '%v'", branch, remote)
		return
	}

	// Create the local branch.
	return Branch(branch, remote+"/"+branch)
}

func CreateOrResetBranch(branch, target string) (stderr *bytes.Buffer, err error) {
	exists, stderr, err := LocalBranchExists(branch)
	if err != nil {
		return
	}

	// Reset the branch in case it exists.
	if exists {
		return ResetKeep(branch, target)
	}
	// Otherwise create a new branch.
	return Branch(branch, target)
}

func Checkout(branch string) (stderr *bytes.Buffer, err error) {
	_, stderr, err = Git("checkout", branch)
	return
}

func ResetKeep(branch, ref string) (stderr *bytes.Buffer, err error) {
	stderr, err = Checkout(branch)
	if err != nil {
		return
	}

	_, stderr, err = Git("reset", "--keep", ref)
	return
}

func ShowByBranch(branch, file string) (content, stderr *bytes.Buffer, err error) {
	return Git("show", branch+":"+file)
}

func Tag(args ...string) (stderr *bytes.Buffer, err error) {
	argsList := append([]string{"tag"}, args...)
	_, stderr, err = Git(argsList...)
	return
}

func DeleteTag(tag string) (stderr *bytes.Buffer, err error) {
	_, stderr, err = Git("tag", "-d", tag)
	return
}

func Hexsha(ref string) (hexsha string, stderr *bytes.Buffer, err error) {
	stdout, stderr, err := Git("show-ref", "--verify", ref)
	if err != nil {
		return
	}

	hexsha = strings.Split(stdout.String(), " ")[0]
	return
}

func EnsureBranchSynchronized(branch, remote string) (stderr *bytes.Buffer, err error) {
	var (
		localRef  = "refs/heads/" + branch
		remoteRef = "refs/remotes/" + remote + "/" + branch
	)
	localHexsha, stderr, err := Hexsha(localRef)
	if err != nil {
		return
	}
	remoteHexsha, stderr, err := Hexsha(remoteRef)
	if err != nil {
		return
	}

	if localHexsha != remoteHexsha {
		err = fmt.Errorf("branch '%v' is not up to date", branch)
	}
	return
}

func EnsureCleanWorkingTree() (status *bytes.Buffer, stderr *bytes.Buffer, err error) {
	status, stderr, err = Git("status", "--porcelain")
	if status.Len() != 0 {
		err = ErrDirtyRepository
	}
	return
}

func CurrentBranch() (branch string, stderr *bytes.Buffer, err error) {
	stdout, stderr, err := Git("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return
	}

	branch = string(bytes.TrimSpace(stdout.Bytes()))
	return
}

func RepositoryRootAbsolutePath() (path string, stderr *bytes.Buffer, err error) {
	stdout, stderr, err := Git("rev-parse", "--show-toplevel")
	if err != nil {
		return
	}

	path = string(bytes.TrimSpace(stdout.Bytes()))
	return
}

func Git(args ...string) (stdout, stderr *bytes.Buffer, err error) {
	stdout = new(bytes.Buffer)
	stderr = new(bytes.Buffer)
	args = append([]string{"--no-pager"}, args...)
	cmd := exec.Command("git", args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err = cmd.Run()
	return
}
