package version

import (
	// Stdlib
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	// Internal
	"github.com/salsita/salsaflow/errs"
	"github.com/salsita/salsaflow/git"
	"github.com/salsita/salsaflow/git/gitutil"
)

const (
	PackageFileName    = "package.json"
	GroupMatcherString = "([0-9]+)[.]([0-9]+)[.]([0-9]+)"
	MatcherString      = "[0-9]+[.][0-9]+[.][0-9]+"
)

type packageFile struct {
	Version string
}

type Version struct {
	Major uint
	Minor uint
	Patch uint
}

func ReadFromBranch(branch string) (ver *Version, stderr *bytes.Buffer, err error) {
	content, err := gitutil.ShowFileByBranch(PackageFileName, branch)
	if err != nil {
		return
	}

	var pkg packageFile
	err = json.Unmarshal(content.Bytes(), &pkg)
	if err != nil {
		return
	}
	if pkg.Version == "" {
		err = fmt.Errorf("version key not found in %v", PackageFileName)
		return
	}

	ver, err = Parse(pkg.Version)
	return
}

func (ver *Version) Zero() bool {
	return ver.Major == 0 && ver.Minor == 0 && ver.Patch == 0
}

func (ver *Version) IncrementMinor() *Version {
	return &Version{ver.Major, ver.Minor + 1, 0}
}

func (ver *Version) IncrementPatch() *Version {
	return &Version{ver.Major, ver.Minor, ver.Patch + 1}
}

func (ver *Version) Set(versionString string) error {
	newVer, err := Parse(versionString)
	if err != nil {
		return err
	}
	ver.Major = newVer.Major
	ver.Minor = newVer.Minor
	ver.Patch = newVer.Patch
	return nil
}

func (ver *Version) String() string {
	return fmt.Sprintf("%v.%v.%v", ver.Major, ver.Minor, ver.Patch)
}

func (ver *Version) ReleaseTagString() string {
	return "v" + ver.String()
}

func (ver *Version) CommitToBranch(branch string) (stderr *bytes.Buffer, err error) {
	// Make sure package.json is clean.
	stderr, err = git.EnsureFileClean(PackageFileName)
	if err != nil {
		return
	}

	// Checkout the branch.
	stderr, err = git.Checkout(branch)
	if err != nil {
		return
	}

	// Get the absolute path of package.json
	root, stderr, err := gitutil.RepositoryRootAbsolutePath()
	if err != nil {
		return
	}
	absPath := filepath.Join(root, PackageFileName)

	// Read package.json
	file, err := os.OpenFile(absPath, os.O_RDWR, 0)
	if err != nil {
		return
	}
	defer file.Close()

	content, err := ioutil.ReadAll(file)
	if err != nil {
		return
	}

	// Parse and replace stuff in package.json
	pattern := regexp.MustCompile(fmt.Sprintf("\"version\": \"%v\"", MatcherString))
	newContent := pattern.ReplaceAllLiteral(content,
		[]byte(fmt.Sprintf("\"version\": \"%v\"", ver)))
	if bytes.Equal(content, newContent) {
		err = fmt.Errorf("%v: failed to replace version string", PackageFileName)
		return
	}

	// Write package.json
	_, err = file.Seek(0, os.SEEK_SET)
	if err != nil {
		return
	}
	err = file.Truncate(0)
	if err != nil {
		return
	}
	_, err = io.Copy(file, bytes.NewReader(newContent))
	if err != nil {
		return
	}
	defer func() {
		if err == nil {
			return
		}
		// On error, checkout package.json to cancel the changes.
		//
		// We cannot lose any changes by doing so, because we make sure that
		// package.json is clean at the beginning of CommitToBranch.
		if stderr, err := git.Checkout("--", absPath); err != nil {
			errs.LogError(fmt.Sprintf("Roll back changes to %v", PackageFileName), err, stderr)
		}
	}()

	// Commit package.json
	stderr, err = git.Add(absPath)
	if err != nil {
		return
	}

	_, stderr, err = git.Run("commit", "-m", fmt.Sprintf("Bump version to %v", ver))
	return
}

func Parse(versionString string) (ver *Version, err error) {
	pattern := regexp.MustCompile("^" + GroupMatcherString + "$")
	parts := pattern.FindStringSubmatch(versionString)
	if len(parts) != 4 {
		return nil, fmt.Errorf("invalid version string: %v", versionString)
	}

	// regexp passed, we know that we are not going to fail here.
	major, _ := strconv.ParseUint(parts[1], 10, 32)
	minor, _ := strconv.ParseUint(parts[2], 10, 32)
	patch, _ := strconv.ParseUint(parts[3], 10, 32)

	return &Version{uint(major), uint(minor), uint(patch)}, nil
}
