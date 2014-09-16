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
	"github.com/salsita/SalsaFlow/git-trunk/git"
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
	content, stderr, err := git.ShowByBranch(branch, PackageFileName)
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

	ver, err = parseVersion(pkg.Version)
	return
}

func (ver *Version) Zero() bool {
	return ver.Major == 0 && ver.Minor == 0 && ver.Patch == 0
}

func (ver *Version) IncrementPatch() *Version {
	return &Version{ver.Major, ver.Minor, ver.Patch + 1}
}

func (ver *Version) Set(versionString string) error {
	newVer, err := parseVersion(versionString)
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
	// Checkout the branch.
	stderr, err = git.Checkout(branch)
	if err != nil {
		return
	}

	// Get the absolute path of package.json
	root, stderr, err := git.RepositoryRootAbsolutePath()
	if err != nil {
		return
	}
	path := filepath.Join(root, PackageFileName)

	// Read package.json
	file, err := os.OpenFile(path, os.O_RDWR, 0)
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

	// Commit package.json
	_, stderr, err = git.Git("add", path)
	if err != nil {
		return
	}
	// XXX: Somehow unstage package.json?
	_, stderr, err = git.Git("commit", "-m", fmt.Sprintf("Bump version to %v", ver))
	return
}

func parseVersion(versionString string) (ver *Version, err error) {
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
