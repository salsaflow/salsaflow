package runners

import "os/exec"

var bashScriptRunner = &ScriptRunner{
	Extension: "bash",
	SupportedPlatforms: []Platform{
		PlatformUnix,
	},
	NewCommand: func(script string) *exec.Cmd {
		return exec.Command("bash", script)
	},
}
