package runners

import "os/exec"

var batchScriptRunner = &ScriptRunner{
	Extension: "bat",
	SupportedPlatforms: []Platform{
		PlatformWindows,
	},
	NewCommand: func(script string) *exec.Cmd {
		return exec.Command("cmd.exe", "/c", script)
	},
}
