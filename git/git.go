package git

import (
	// Stdlib
	"bufio"
	"bytes"
	"fmt"
	"strings"

	// Internal
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/shell"

	// Internal
	"github.com/salsaflow/salsaflow/git/gitutil"
)

func Add(args ...string) error {
	_, err := RunCommand("add", args...)
	return err
}

func Branch(args ...string) error {
	_, err := RunCommand("branch", args...)
	return err
}

func Checkout(args ...string) error {
	_, err := RunCommand("checkout", args...)
	return err
}

func CherryPick(args ...string) error {
	_, err := RunCommand("cherry-pick", args...)
	return err
}

func Log(args ...string) (stdout *bytes.Buffer, err error) {
	return RunCommand("log", args...)
}

func Rebase(args ...string) error {
	_, err := RunCommand("rebase", args...)
	return err
}

func Reset(args ...string) error {
	_, err := RunCommand("reset", args...)
	return err
}

func Status(args ...string) (stdout *bytes.Buffer, err error) {
	return RunCommand("status", args...)
}

func Tag(args ...string) error {
	_, err := RunCommand("tag", args...)
	return err
}

func DeleteTag(tag string) error {
	return Tag("-d", tag)
}

func Push(remote string, args ...string) error {
	argsList := make([]string, 3, 3+len(args))
	argsList[0], argsList[1], argsList[2] = "push", "-u", remote
	argsList = append(argsList, args...)
	_, err := Run(argsList...)
	return err
}

func PushForce(remote string, args ...string) error {
	argsList := make([]string, 3, 3+len(args))
	argsList[0], argsList[1], argsList[2] = "push", "-f", remote
	argsList = append(argsList, args...)
	_, err := Run(argsList...)
	return err
}

func UpdateRemotes(remotes ...string) error {
	argsList := make([]string, 2, 2+len(remotes))
	argsList[0] = "remote"
	argsList[1] = "update"
	argsList = append(argsList, remotes...)
	_, err := Run(argsList...)
	return err
}

func RefExists(ref string) (exists bool, err error) {
	task := fmt.Sprintf("Run 'git show-ref --quiet %v'", ref)
	_, stderr, err := shell.Run("git", "show-ref", "--quiet", ref)
	if err != nil {
		if stderr.Len() != 0 {
			// Non-empty error output means that there was an error.
			return false, errs.NewError(task, err, stderr)
		}
		// Otherwise the ref does not exist.
		return false, nil
	}
	// No error means that the ref exists.
	return true, nil
}

// RefExistsStrict requires the whole ref path to be specified,
// e.g. refs/remotes/origin/master.
func RefExistsStrict(ref string) (exists bool, err error) {
	task := fmt.Sprintf("Run 'git show-ref --quiet --verify %v'", ref)
	_, stderr, err := shell.Run("git", "show-ref", "--verify", "--quiet", ref)
	if err != nil {
		if stderr.Len() != 0 {
			// Non-empty error output means that there was an error.
			return false, errs.NewError(task, err, stderr)
		}
		// Otherwise the ref does not exist.
		return false, nil
	}
	// No error means that the ref exists.
	return true, nil
}

func EnsureBranchNotExist(branch string, remote string) error {
	exists, err := LocalBranchExists(branch)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("branch '%v' already exists", branch)
	}

	exists, err = RemoteBranchExists(branch, remote)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("branch '%v' already exists in remote '%v'", branch, remote)
	}

	return nil
}

func LocalBranchExists(branch string) (exists bool, err error) {
	return RefExistsStrict("refs/heads/" + branch)
}

func RemoteBranchExists(branch string, remote string) (exists bool, err error) {
	return RefExistsStrict(fmt.Sprintf("refs/remotes/%v/%v", remote, branch))
}

func CreateTrackingBranchUnlessExists(branch string, remote string) error {
	// Check whether the local branch exists and just return in that case.
	exists, err := LocalBranchExists(branch)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	// Check whether the remote counterpart exists.
	exists, err = RemoteBranchExists(branch, remote)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("branch '%v' not found in the remote '%v'", branch, remote)
	}

	// Create the local branch.
	return Branch(branch, remote+"/"+branch)
}

func CreateOrResetBranch(branch, target string) (action.Action, error) {
	exists, err := LocalBranchExists(branch)
	if err != nil {
		return nil, err
	}
	// Reset the branch in case it exists.
	if exists {
		return resetBranch(branch, target)
	}
	// Otherwise create a new branch.
	return createBranch(branch, target)
}

func resetBranch(branch, target string) (action.Action, error) {
	// Remember the current position.
	current, err := Hexsha("refs/heads/" + branch)
	if err != nil {
		return nil, err
	}

	// Reset the branch.
	if err := ResetKeep(branch, target); err != nil {
		return nil, err
	}

	return action.ActionFunc(func() error {
		// On rollback, reset the branch to the original position.
		return ResetKeep(branch, current)
	}), nil
}

func createBranch(branch, target string) (action.Action, error) {
	// Create the branch.
	if err := Branch(branch, target); err != nil {
		return nil, err
	}

	return action.ActionFunc(func() error {
		// On rollback, delete the branch.
		return Branch("-D", branch)
	}), nil
}

func ResetKeep(branch, ref string) (err error) {
	// Remember the current branch.
	currentBranch, err := CurrentBranch()
	if err != nil {
		return err
	}

	// Checkout the branch to be reset.
	if err := Checkout(branch); err != nil {
		return err
	}
	defer func() {
		// Checkout the original branch on return.
		if ex := Checkout(currentBranch); ex != nil {
			if err == nil {
				err = ex
			} else {
				errs.Log(ex)
			}
		}
	}()

	// Reset the branch.
	_, err = Run("reset", "--keep", ref)
	return err
}

func Hexsha(ref string) (hexsha string, err error) {
	stdout, err := Run("show-ref", "--verify", ref)
	if err != nil {
		return "", err
	}

	return strings.Split(stdout.String(), " ")[0], nil
}

func EnsureBranchSynchronized(branch, remote string) error {
	exists, err := RemoteBranchExists(branch, remote)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	var (
		localRef  = "refs/heads/" + branch
		remoteRef = "refs/remotes/" + remote + "/" + branch
	)
	localHexsha, err := Hexsha(localRef)
	if err != nil {
		return err
	}
	remoteHexsha, err := Hexsha(remoteRef)
	if err != nil {
		return err
	}

	if localHexsha != remoteHexsha {
		return fmt.Errorf("branch '%v' is not up to date", branch)
	}
	return nil
}

func EnsureCleanWorkingTree(includeUntracked bool) error {
	status, err := Run("status", "--porcelain")
	if err != nil {
		return err
	}

	// In case the output is not empty and we include all files,
	// this is enough to say that the repository is dirty.
	if status.Len() != 0 && includeUntracked {
		return ErrDirtyRepository
	}

	scanner := bufio.NewScanner(status)
	for scanner.Scan() {
		// Skip the files that are untracked.
		if strings.HasPrefix(scanner.Text(), "?? ") {
			continue
		}
		// Otherwise the repository is dirty.
		return ErrDirtyRepository
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func EnsureFileClean(relativePath string) error {
	status, err := Run("status", "--porcelain", relativePath)
	if err != nil {
		return err
	}
	if status.Len() != 0 {
		return &ErrDirtyFile{relativePath}
	}
	return nil
}

func CurrentBranch() (branch string, err error) {
	stdout, err := Run("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}

	return string(bytes.TrimSpace(stdout.Bytes())), nil
}

func GetConfigString(key string) (value string, err error) {
	task := fmt.Sprintf("Run 'git config %v'", key)
	stdout, stderr, err := shell.Run("git", "config", key)
	if err != nil {
		if stderr.Len() == 0 {
			// git config returns exit code 1 when the key is not set.
			// This can be detected by stderr being of zero length.
			// We treat this as the key being set to "".
			return "", nil
		}
		// Otherwise there is an error.
		return "", errs.NewError(task, err, stderr)
	}
	// Just return what was printed to stdout.
	return strings.TrimSpace(stdout.String()), nil
}

func SetConfigString(key string, value string) error {
	task := fmt.Sprintf("Run 'git config %v %v'", key, value)
	_, stderr, err := shell.Run("git", "config", key, value)
	if err != nil {
		return errs.NewError(task, err, stderr)
	}
	return nil
}

func Run(args ...string) (stdout *bytes.Buffer, err error) {
	return gitutil.Run(args...)
}

func RunCommand(command string, args ...string) (stdout *bytes.Buffer, err error) {
	return gitutil.RunCommand(command, args...)
}
