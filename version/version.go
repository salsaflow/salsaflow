package version

import (
	// Stdlib
	"errors"
	"fmt"
	"regexp"
	"strconv"

	// Internal
	"github.com/salsaflow/salsaflow/errs"
)

const (
	MatcherString      = "[0-9]+[.][0-9]+[.][0-9]+"
	GroupMatcherString = "([0-9]+)[.]([0-9]+)[.]([0-9]+)"
)

type Version struct {
	Major uint
	Minor uint
	Patch uint
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

func Parse(versionString string) (*Version, error) {
	task := "Parse version string: " + versionString
	pattern := regexp.MustCompile("^" + GroupMatcherString + "$")
	parts := pattern.FindStringSubmatch(versionString)
	if len(parts) != 4 {
		return nil, errs.NewError(task, errors.New("invalid version string: "+versionString), nil)
	}

	// regexp passed, we know that we are not going to fail here.
	major, _ := strconv.ParseUint(parts[1], 10, 32)
	minor, _ := strconv.ParseUint(parts[2], 10, 32)
	patch, _ := strconv.ParseUint(parts[3], 10, 32)

	return &Version{uint(major), uint(minor), uint(patch)}, nil
}

func FromTag(tag string) (*Version, error) {
	return Parse(tag[1:])
}
