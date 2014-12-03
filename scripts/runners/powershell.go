package runners

import "os/exec"

var powerShellScriptRunner = &ScriptRunner{
	Extension: "ps1",
	SupportedPlatforms: []Platform{
		PlatformWindows,
	},
	NewCommand: func(script string) *exec.Cmd {
		return exec.Command("PowerShell.exe", "-NonInteractive", "-File", script)
	},
}
