package runners

import "os/exec"

var nodeScriptRunner = &ScriptRunner{
	Extension: "js",
	SupportedPlatforms: []Platform{
		PlatformUnix,
		PlatformWindows,
	},
	NewCommand: func(script string) *exec.Cmd {
		// Should work just like that, the script always ends with "js".
		return exec.Command("node", script)
	},
}
