package git

import (
	// Stdlib
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	// Internal
	"github.com/salsita/salsaflow/shell"
)

func UpdateRemotes(remotes ...string) (stderr *bytes.Buffer, err error) {
	argsList := append([]string{"remote", "update"}, remotes...)
	_, stderr, err = Git(argsList...)
	return
}

func Push(remote string, args ...string) (stderr *bytes.Buffer, err error) {
	argsList := make([]string, 2, 2+len(args))
	argsList[0] = "push"
	argsList[1] = remote
	argsList = append(argsList, args...)
	_, stderr, err = Git(argsList...)
	return
}

func PushForce(remote string, args ...string) (stderr *bytes.Buffer, err error) {
	argsList := make([]string, 3, 3+len(args))
	argsList[0] = "push"
	argsList[1] = "-f"
	argsList[2] = remote
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
	if err != nil {
		return nil, stderr, err
	}
	if status.Len() != 0 {
		return status, nil, ErrDirtyRepository
	}
	return nil, nil, nil
}

func EnsureFileClean(relativePath string) (stderr *bytes.Buffer, err error) {
	status, stderr, err := Git("status", "--porcelain", relativePath)
	if err != nil {
		return stderr, err
	}
	if status.Len() != 0 {
		return nil, &ErrDirtyFile{relativePath}
	}
	return nil, nil
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

// RelativePath returns the relative path from the current working directory to the file
// specified by the relative path from the repository root.
//
// This is useful for some other Git commands, particularly git status.
func RelativePath(pathFromRoot string) (relativePath string, stderr *bytes.Buffer, err error) {
	root, stderr, err := RepositoryRootAbsolutePath()
	if err != nil {
		return "", stderr, err
	}
	absolutePath := filepath.Join(root, pathFromRoot)

	cwd, err := os.Getwd()
	if err != nil {
		return "", nil, err
	}

	relativePath, err = filepath.Rel(cwd, absolutePath)
	if err != nil {
		return "", nil, err
	}

	return relativePath, nil, nil
}

func GetConfigBool(key string) (value bool, stderr *bytes.Buffer, err error) {
	stdout, stderr, err := Git("config", key)
	if err != nil {
		if stderr.Len() == 0 {
			// git config returns exit code 1 when the key is not set.
			// This can be detected by stderr being of zero length.
			// We treat this as the key being set to false.
			return false, nil, nil
		}
		// Otherwise there is an error.
		return false, stderr, err
	}
	// Otherwise a boolean value is written into stdout, so we parse it.
	v, err := strconv.ParseBool(strings.TrimSpace(stdout.String()))
	if err != nil {
		return false, nil, err
	}
	return v, nil, nil
}

func SetConfigBool(key string, value bool) (stderr *bytes.Buffer, err error) {
	_, stderr, err = Git("config", key, strconv.FormatBool(value))
	return
}

func Git(args ...string) (stdout, stderr *bytes.Buffer, err error) {
	args = append([]string{"git", "--no-pager"}, args...)
	return shell.Run(args...)
}
