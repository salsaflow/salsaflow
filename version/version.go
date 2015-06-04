package version

import (
	// Vendor
	"github.com/blang/semver"
)

const (
	MatcherString      = "[0-9]+[.][0-9]+[.][0-9]+"
	GroupMatcherString = "([0-9]+)[.]([0-9]+)[.]([0-9]+)"
)

type Version struct {
	semver.Version
}

func (v *Version) Zero() bool {
	return v.Major == 0 && v.Minor == 0 && v.Patch == 0 && len(v.Pre) == 0 && len(v.Build) == 0
}

func (v *Version) IncrementMinor() *Version {
	return &Version{semver.Version{
		Major: v.Major,
		Minor: v.Minor + 1,
	}}
}

func (v *Version) IncrementPatch() *Version {
	return &Version{semver.Version{
		Major: v.Major,
		Minor: v.Minor,
		Patch: v.Patch + 1,
	}}
}

func (v *Version) ReleaseTagString() string {
	return "v" + v.String()
}

func Parse(versionString string) (*Version, error) {
	v, err := semver.Parse(versionString)
	if err != nil {
		return nil, err
	}
	return &Version{v}, nil
}

func FromTag(tag string) (*Version, error) {
	return Parse(tag[1:])
}

// Set implements flag.Value interface.
func (v *Version) Set(versionString string) error {
	ver, err := Parse(versionString)
	if err != nil {
		return err
	}
	v.Version = ver.Version
	return nil
}
