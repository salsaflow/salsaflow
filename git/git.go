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
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/shell"

	// Internal
	"github.com/salsaflow/salsaflow/git/gitutil"
	"github.com/salsaflow/salsaflow/prompt"
)

const ZeroHash = "0000000000000000000000000000000000000000"

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

func Merge(args ...string) error {
	_, err := RunCommand("merge", args...)
	return err
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
	argsList := make([]string, 4, 4+len(args))
	argsList[0], argsList[1], argsList[2], argsList[3] = "push", "-u", "-f", remote
	argsList = append(argsList, args...)
	_, err := Run(argsList...)
	return err
}

func UpdateRemotes(remotes ...string) error {
	argsList := make([]string, 3, 3+len(remotes))
	argsList[0] = "remote"
	argsList[1] = "update"
	argsList[2] = "--prune"
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
			return false, errs.NewErrorWithHint(task, err, stderr.String())
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
			return false, errs.NewErrorWithHint(task, err, stderr.String())
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

func CreateOrResetBranch(branch, target string) (action.Action, error) {
	// Make sure the target exists.
	// We do this manually so that we can return *ErrRefNotFound.
	exists, err := RefExists(target)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, &ErrRefNotFound{target}
	}

	// Decide what to do next.
	exists, err = LocalBranchExists(branch)
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
	if err := SetBranch(branch, target); err != nil {
		return nil, err
	}

	return action.ActionFunc(func() error {
		// On rollback, reset the branch to the original position.
		return SetBranch(branch, current)
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

func SetBranch(branch, targetRef string) error {
	return Branch("-f", branch, targetRef)
}

func Hexsha(ref string) (hexsha string, err error) {
	stdout, err := Run("show-ref", "--verify", ref)
	if err != nil {
		return "", err
	}

	return strings.Split(stdout.String(), " ")[0], nil
}

func BranchHexsha(branch string) (hexsha string, err error) {
	return Hexsha("refs/heads/" + branch)
}

// IsBranchSynchronized returns true when the branch of the given name
// is up to date with the given remote. In case the branch does not exist
// in the remote repository, true is returned as well.
func IsBranchSynchronized(branch, remote string) (bool, error) {
	exists, err := RemoteBranchExists(branch, remote)
	if err != nil {
		return false, err
	}
	if !exists {
		return true, nil
	}

	var (
		localRef  = "refs/heads/" + branch
		remoteRef = "refs/remotes/" + remote + "/" + branch
	)
	localHexsha, err := Hexsha(localRef)
	if err != nil {
		return false, err
	}
	remoteHexsha, err := Hexsha(remoteRef)
	if err != nil {
		return false, err
	}

	return localHexsha == remoteHexsha, nil
}

// EnsureBranchSynchronized makes sure the given branch is up to date.
// If that is not the case, *ErrRefNoInSync is returned.
func EnsureBranchSynchronized(branch, remote string) (err error) {
	// Check whether the remote counterpart actually exists.
	exists, err := RemoteBranchExists(branch, remote)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	// Get the data needed.
	var (
		localRef  = fmt.Sprintf("refs/heads/%v", branch)
		remoteRef = fmt.Sprintf("refs/remotes/%v/%v", remote, branch)
	)
	localHexsha, err := Hexsha(localRef)
	if err != nil {
		return err
	}
	remoteHexsha, err := Hexsha(remoteRef)
	if err != nil {
		return err
	}
	if localHexsha == remoteHexsha {
		// The branch is up to date, we are done here.
		return nil
	}

	// Check whether the local branch can be fast-forwarded.
	remoteBranch := fmt.Sprintf("%v/%v", remote, branch)
	_, err = Run("merge-base", "--is-ancestor", branch, remoteBranch)
	if err != nil {
		// --is-ancestor returns exit status 0 on true, 1 on false, some other on error.
		// We cannot check the value in a platform-independent way, but we count on the fact that
		// stderr will be non-empty on error.
		ex, ok := err.(errs.Err)
		if !ok || len(ex.Hint()) != 0 {
			// In case err is not implementing errs.Err or len(stderr) != 0, we return the error.
			return err
		}
		// Otherwise the error means that --is-ancestor returned false,
		// so we cannot fast-forward and we have to return an error.
		return &ErrRefNotInSync{branch}
	}

	// Perform a fast-forward merge.
	// Ask the user before doing so.
	fmt.Println()
	fmt.Printf("Branch '%v' is behind '%v', and can be fast-forwarded.\n", branch, remoteBranch)
	proceed, err := prompt.Confirm("Shall we perform the merge? It's all safe!", true)
	fmt.Println()
	if err != nil {
		return err
	}
	if !proceed {
		return &ErrRefNotInSync{branch}
	}

	// Make sure the right branch is checked out.
	currentBranch, err := gitutil.CurrentBranch()
	if err != nil {
		return err
	}
	if branch != currentBranch {
		// Checkout the branch to be merged.
		task := fmt.Sprintf("Checkout branch '%v'", branch)
		if err := Checkout(branch); err != nil {
			return errs.NewError(task, err)
		}
		defer func() {
			// Checkout the original branch on return.
			task := fmt.Sprintf("Checkout branch '%v'", currentBranch)
			if ex := Checkout(currentBranch); ex != nil {
				if err == nil {
					err = ex
				} else {
					errs.LogError(task, err)
				}
			}
		}()
	}

	// Merge. Use --ff-only, just to be sure.
	// But we have already checked that this will be a fast-forward merge.
	_, err = Run("merge", "--ff-only")
	if err != nil {
		return err
	}

	log.Log(fmt.Sprintf("Branch '%v' fast-forwarded onto '%v'", branch, remoteBranch))
	return nil
}

// CheckOrCreateTrackingBranch tries to make sure that a local branch
// of the given name exists and is in sync with the given remote.
//
// So, in case the right remote branch exists and the local does not,
// the local tracking branch is created. In case the local branch
// exists already, it is ensured that it is up to date.
//
// In case the remote branch does not exist, *ErrRefNotFound is returned.
// In case the branch is not up to date, *ErrRefNotInSync is returned.
func CheckOrCreateTrackingBranch(branch, remote string) error {
	// Get the data on the local branch.
	localExists, err := LocalBranchExists(branch)
	if err != nil {
		return err
	}

	// Check whether the remote counterpart exists.
	remoteExists, err := RemoteBranchExists(branch, remote)
	if err != nil {
		return err
	}
	if !remoteExists {
		if localExists {
			log.Warn(fmt.Sprintf(
				"Local branch '%v' found, but the remote counterpart is missing", branch))
			log.NewLine(fmt.Sprintf(
				"Please delete or push local branch '%v'", branch))
		}
		return &ErrRefNotFound{remote + "/" + branch}
	}

	// Check whether the local branch exists.
	if !localExists {
		if err := Branch(branch, remote+"/"+branch); err != nil {
			return err
		}
		log.Log(fmt.Sprintf("Local branch '%v' created (tracking %v/%v)", branch, remote, branch))
		return nil
	}

	// In case it exists, make sure that it is up to date.
	return EnsureBranchSynchronized(branch, remote)
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
		return "", errs.NewErrorWithHint(task, err, stderr.String())
	}
	// Just return what was printed to stdout.
	return strings.TrimSpace(stdout.String()), nil
}

func SetConfigString(key string, value string) error {
	task := fmt.Sprintf("Run 'git config %v %v'", key, value)
	_, stderr, err := shell.Run("git", "config", key, value)
	if err != nil {
		return errs.NewErrorWithHint(task, err, stderr.String())
	}
	return nil
}

func Run(args ...string) (stdout *bytes.Buffer, err error) {
	return gitutil.Run(args...)
}

func RunCommand(command string, args ...string) (stdout *bytes.Buffer, err error) {
	return gitutil.RunCommand(command, args...)
}
