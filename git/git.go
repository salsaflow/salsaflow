package git

import (
	// Stdlib
	"bytes"
	"fmt"
	"strings"

	// Internal
	"github.com/salsita/salsaflow/errs"

	// Internal
	"github.com/salsita/salsaflow/git/gitutil"
)

func Add(args ...string) (stderr *bytes.Buffer, err error) {
	_, stderr, err = RunCommand("add", args...)
	return
}

func Branch(args ...string) (stderr *bytes.Buffer, err error) {
	_, stderr, err = RunCommand("branch", args...)
	return stderr, err
}

func Checkout(args ...string) (stderr *bytes.Buffer, err error) {
	_, stderr, err = RunCommand("checkout", args...)
	return
}

func CherryPick(args ...string) (stderr *bytes.Buffer, err error) {
	_, stderr, err = RunCommand("cherry-pick", args...)
	return
}

func Log(args ...string) (stdout, stderr *bytes.Buffer, err error) {
	return RunCommand("log", args...)
}

func Rebase(args ...string) (stderr *bytes.Buffer, err error) {
	_, stderr, err = RunCommand("rebase", args...)
	return
}

func Status(args ...string) (stdout, stderr *bytes.Buffer, err error) {
	return RunCommand("status", args...)
}

func Tag(args ...string) (stderr *bytes.Buffer, err error) {
	_, stderr, err = RunCommand("tag", args...)
	return stderr, err
}

func DeleteTag(tag string) (stderr *bytes.Buffer, err error) {
	return Tag("-d", tag)
}

func Push(remote string, args ...string) (stderr *bytes.Buffer, err error) {
	argsList := make([]string, 3, 3+len(args))
	argsList[0], argsList[1], argsList[2] = "push", "-u", remote
	argsList = append(argsList, args...)
	_, stderr, err = Run(argsList...)
	return
}

func PushForce(remote string, args ...string) (stderr *bytes.Buffer, err error) {
	argsList := make([]string, 3, 3+len(args))
	argsList[0], argsList[1], argsList[2] = "push", "-f", remote
	argsList = append(argsList, args...)
	_, stderr, err = Run(argsList...)
	return
}

func UpdateRemotes(remotes ...string) (stderr *bytes.Buffer, err error) {
	argsList := append([]string{"remote", "update"}, remotes...)
	_, stderr, err = Run(argsList...)
	return
}

func RefExists(ref string) (exists bool, stderr *bytes.Buffer, err error) {
	_, out, err := Run("show-ref", "--quiet", ref)
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
	_, out, err := Run("show-ref", "--verify", "--quiet", ref)
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

func ResetKeep(branch, ref string) (stderr *bytes.Buffer, err error) {
	stderr, err = Checkout(branch)
	if err != nil {
		return
	}

	_, stderr, err = Run("reset", "--keep", ref)
	return
}

func Hexsha(ref string) (hexsha string, stderr *bytes.Buffer, err error) {
	stdout, stderr, err := Run("show-ref", "--verify", ref)
	if err != nil {
		return
	}

	hexsha = strings.Split(stdout.String(), " ")[0]
	return
}

func EnsureBranchSynchronized(branch, remote string) (stderr *bytes.Buffer, err error) {
	exists, stderr, err := RemoteBranchExists(branch, remote)
	if err != nil {
		return stderr, err
	}
	if !exists {
		return nil, nil
	}

	var (
		localRef  = "refs/heads/" + branch
		remoteRef = "refs/remotes/" + remote + "/" + branch
	)
	localHexsha, stderr, err := Hexsha(localRef)
	if err != nil {
		return stderr, err
	}
	remoteHexsha, stderr, err := Hexsha(remoteRef)
	if err != nil {
		return stderr, err
	}

	if localHexsha != remoteHexsha {
		return stderr, fmt.Errorf("branch '%v' is not up to date", branch)
	}
	return nil, nil
}

func EnsureCleanWorkingTree() (status *bytes.Buffer, stderr *bytes.Buffer, err error) {
	status, stderr, err = Run("status", "--porcelain")
	if err != nil {
		return nil, stderr, err
	}
	if status.Len() != 0 {
		return status, nil, ErrDirtyRepository
	}
	return nil, nil, nil
}

func EnsureFileClean(relativePath string) (stderr *bytes.Buffer, err error) {
	status, stderr, err := Run("status", "--porcelain", relativePath)
	if err != nil {
		return stderr, err
	}
	if status.Len() != 0 {
		return nil, &ErrDirtyFile{relativePath}
	}
	return nil, nil
}

func CurrentBranch() (branch string, stderr *bytes.Buffer, err error) {
	stdout, stderr, err := Run("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return
	}

	branch = string(bytes.TrimSpace(stdout.Bytes()))
	return
}

func GetConfigString(key string) (value string, err error) {
	task := fmt.Sprintf("Run 'git config %v'", key)
	stdout, stderr, err := Run("config", key)
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
	_, stderr, err := Run("config", key, value)
	if err != nil {
		return errs.NewError(task, err, stderr)
	}
	return nil
}

func Run(args ...string) (stdout, stderr *bytes.Buffer, err error) {
	return gitutil.Run(args...)
}

func RunCommand(command string, args ...string) (stdout, stderr *bytes.Buffer, err error) {
	return gitutil.RunCommand(command, args...)
}
