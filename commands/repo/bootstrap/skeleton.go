package bootstrapCmd

import (
	// Stdlib
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	// Internal
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/git"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/shell"
)

const SkeletonCacheDirname = ".salsaflow_skeletons"

func getOrUpdateSkeleton(skeleton string) error {
	// Parse the skeleton string.
	parts := strings.SplitN(skeleton, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("not a valid repository path string: %v", skeleton)
	}
	owner, repo := parts[0], parts[1]

	// Create the cache directory if necessary.
	task := "Make sure the local cache directory exists"
	cacheDir, err := cacheDirectoryAbsolutePath()
	if err != nil {
		return errs.NewError(task, err)
	}
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return errs.NewError(task, err)
	}

	// Pull or close the given skeleton.
	task = "Pull or clone the given skeleton"
	skeletonDir := filepath.Join(cacheDir, "github.com", owner)

	if err := os.MkdirAll(skeletonDir, 0755); err != nil {
		return errs.NewError(task, err)
	}

	skeletonPath := filepath.Join(skeletonDir, repo)
	if _, err := os.Stat(skeletonPath); err != nil {
		if !os.IsNotExist(err) {
			return errs.NewError(task, err)
		}

		// The directory does not exist, hence we clone.
		task := fmt.Sprintf("Clone skeleton '%v'", skeleton)
		log.Run(task)
		args := []string{
			"clone",
			"--single-branch",
			fmt.Sprintf("https://github.com/%v/%v", owner, repo),
			skeletonPath,
		}
		if _, err := git.Run(args...); err != nil {
			return errs.NewError(task, err)
		}
		return nil
	}

	// The skeleton directory exists, hence we pull.
	task = fmt.Sprintf("Pull skeleton '%v'", skeleton)
	log.Run(task)
	cmd, _, stderr := shell.Command("git", "pull")
	cmd.Dir = skeletonPath
	if err := cmd.Run(); err != nil {
		return errs.NewErrorWithHint(task, err, stderr.String())
	}
	return nil
}

// pourSkeleton counts on the fact that skeleton is a valid skeleton
// that is available in the local cache directory.
func pourSkeleton(skeletonName string, localConfigDir string) (err error) {
	// Get the skeleton src path.
	cacheDir, err := cacheDirectoryAbsolutePath()
	if err != nil {
		return err
	}
	skeletonDir := filepath.Join(cacheDir, "github.com", skeletonName)

	// Make sure src is a directory, just to be sure.
	skeletonInfo, err := os.Stat(skeletonDir)
	if err != nil {
		return err
	}
	if !skeletonInfo.IsDir() {
		return fmt.Errorf("skeleton source path not a directory: %v", skeletonDir)
	}

	// Get the list of script files.
	srcScriptsDir := filepath.Join(skeletonDir, "scripts")
	scripts, err := filepath.Glob(srcScriptsDir + "/*")
	if err != nil {
		return err
	}
	if len(scripts) == 0 {
		log.Warn("No script files found in the skeleton repository")
		return nil
	}

	// Create the destination directory.
	dstScriptsDir := filepath.Join(localConfigDir, "scripts")
	if err := os.MkdirAll(dstScriptsDir, 0755); err != nil {
		return err
	}
	// Delete the directory on error.
	defer action.RollbackOnError(&err, action.ActionFunc(func() error {
		log.Rollback("Create the local scripts directory")
		if err := os.RemoveAll(dstScriptsDir); err != nil {
			return errs.NewError("Remove the local scripts directory", err)
		}
		return nil
	}))

	for _, script := range scripts {
		err := func(script string) error {
			// Skip directories.
			scriptInfo, err := os.Stat(script)
			if err != nil {
				return err
			}
			if scriptInfo.IsDir() {
				return nil
			}

			// Copy the file.
			filename := script[len(srcScriptsDir)+1:]
			fmt.Println("---> Copy", filepath.Join("scripts", filename))
			return nil

			// Otherwise just copy the file.
			srcFd, err := os.Open(script)
			if err != nil {
				return err
			}
			defer srcFd.Close()

			dstPath := filepath.Join(dstScriptsDir, filename)
			dstFd, err := os.OpenFile(
				dstPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, scriptInfo.Mode())
			if err != nil {
				return err
			}
			defer dstFd.Close()

			_, err = io.Copy(dstFd, srcFd)
			return err
		}(script)
		if err != nil {
			return err
		}

	}

	return nil
}

func cacheDirectoryAbsolutePath() (path string, err error) {
	me, err := user.Current()
	if err != nil {
		return "", nil
	}
	return filepath.Join(me.HomeDir, SkeletonCacheDirname), nil
}
