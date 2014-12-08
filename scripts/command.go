package scripts

import (
	// Stdlib
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	// Internal
	"github.com/salsaflow/salsaflow/config"
	"github.com/salsaflow/salsaflow/git/gitutil"
	"github.com/salsaflow/salsaflow/scripts/runners"
)

const ScriptDirname = "scripts"

type ErrNotFound struct {
	scriptName string
}

func (err *ErrNotFound) Error() string {
	return fmt.Sprintf("no custom SalsaFlow script available for '%v'", err.scriptName)
}

func (err *ErrNotFound) ScriptName() string {
	return err.scriptName
}

// Commands returns *exec.Command for the given script name and args.
func Command(scriptName string, args ...string) (*exec.Cmd, error) {
	// Make sure this is a script name, not a path.
	if strings.Contains(scriptName, "/") {
		return nil, fmt.Errorf("not a script name: %v", scriptName)
	}

	// Get the list of available scripts.
	root, err := gitutil.RepositoryRootAbsolutePath()
	if err != nil {
		return nil, err
	}

	scriptsDirPath := filepath.Join(root, config.LocalConfigDirname, ScriptDirname)

	scriptsDir, err := os.Open(scriptsDirPath)
	if err != nil {
		return nil, err
	}
	defer scriptsDir.Close()

	scripts, err := scriptsDir.Readdirnames(-1)
	if err != nil {
		return nil, err
	}

	// Choose the first available script for the given script name.
	// Return a command for that script based on the file extension and platform.
	currentPlatformId := runners.ParsePlatform(runtime.GOOS)

	for _, script := range scripts {
		// To understand the loop:
		//   script     = ${base}.${ext}
		//   base       = ${name}_${platform}
		//   (=> script = ${name}_${platform}.${ext})

		// Get the file extension.
		ext := filepath.Ext(script)
		if ext == "" {
			continue
		}

		// Get the file base, i.e. the filename without the file extension.
		base := script[:len(script)-len(ext)]

		// In case the whole base matches, this is a cross-platform script and we are done.
		// Otherwise we have to parse the base further and decide what to do.
		var platformId runners.Platform
		if base == scriptName {
			platformId = currentPlatformId
		} else {
			// Split the base.
			parts := strings.Split(base, "_")

			// Make sure the platform matches the current platform.
			platform := parts[len(parts)-1]
			platformId = runners.ParsePlatform(platform)
			if platformId != currentPlatformId {
				continue
			}

			// Make sure the name matches the requested script name.
			name := base[:len(base)-len(platform)-1]
			if name != scriptName {
				continue
			}
		}

		// Get the runner for the given file extension.
		// ext contains the dot, which we need to drop.
		runner := runners.GetRunner(ext[1:], platformId)
		if runner == nil {
			continue
		}
		cmd := runner.NewCommand(filepath.Join(scriptsDirPath, script))
		cmd.Args = append(cmd.Args, args...)
		cmd.Dir = root
		return cmd, nil
	}

	return nil, &ErrNotFound{scriptName}
}
