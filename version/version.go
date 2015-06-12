package version

import (
	// Stdlib
	"fmt"

	// Vendor
	"github.com/blang/semver"
)

type Version struct {
	semver.Version
}

// BaseString only returns MAJOR.MINOR.PATCH
func (v *Version) BaseString() string {
	return fmt.Sprintf("%v.%v.%v", v.Major, v.Minor, v.Patch)
}

func (v *Version) Clone() *Version {
	return &Version{v.Version}
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

type versionKind int

const (
	vkTrunk versionKind = iota
	vkTesting
	vkStage
	vkStable
)

func (v *Version) ToTrunkVersion() (*Version, error) {
	return v.toVersion(vkTrunk)
}

func (v *Version) ToTestingVersion() (*Version, error) {
	return v.toVersion(vkTesting)
}

func (v *Version) ToStageVersion() (*Version, error) {
	return v.toVersion(vkStage)
}

func (v *Version) ToStableVersion() (*Version, error) {
	return v.toVersion(vkStable)
}

func (v *Version) toVersion(kind versionKind) (*Version, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	ver := v.Clone()
	ver.Build = nil

	switch kind {
	case vkTrunk:
		ver.Pre = []semver.PRVersion{config.TrunkSuffix()}
	case vkTesting:
		ver.Pre = []semver.PRVersion{config.TestingSuffix()}
	case vkStage:
		ver.Pre = []semver.PRVersion{config.StageSuffix()}
	case vkStable:
		ver.Pre = nil
	default:
		panic(fmt.Errorf("not a valid version kind: %v", kind))
	}

	return ver, nil
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
