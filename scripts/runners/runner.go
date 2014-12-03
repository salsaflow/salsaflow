package runners

import (
	"os/exec"
)

type Platform int

const (
	PlatformUnix Platform = iota + 1
	PlatformWindows
	PlatformUnsupported
)

type ScriptRunner struct {
	Extension          string
	SupportedPlatforms []Platform
	NewCommand         func(script string) *exec.Cmd
}

var runners = []*ScriptRunner{
	bashScriptRunner,
	batchScriptRunner,
	nodeScriptRunner,
	powerShellScriptRunner,
}

func ParsePlatform(platform string) Platform {
	switch platform {
	case "darwin":
		return PlatformUnix
	case "dragonfly":
		return PlatformUnix
	case "freebsd":
		return PlatformUnix
	case "linux":
		return PlatformUnix
	case "netbsd":
		return PlatformUnix
	case "openbsd":
		return PlatformUnix
	case "unix":
		return PlatformUnix
	case "windows":
		return PlatformWindows
	default:
		return PlatformUnsupported
	}
}

func GetRunner(ext string, platform Platform) *ScriptRunner {
	for _, runner := range runners {
		if runner.Extension != ext {
			continue
		}

		for _, p := range runner.SupportedPlatforms {
			if p == platform {
				return runner
			}
		}
	}

	return nil
}
