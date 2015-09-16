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
func pourSkeleton(skeletonName string, localConfigDir string) error {
	// Get the skeleton src path.
	cacheDir, err := cacheDirectoryAbsolutePath()
	if err != nil {
		return err
	}
	src := filepath.Join(cacheDir, "github.com", skeletonName)

	// Make sure src is a directory, just to be sure.
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !srcInfo.IsDir() {
		return fmt.Errorf("skeleton source path not a directory: %v", src)
	}

	// Walk src and copy the files.
	return filepath.Walk(src, func(srcPath string, srcPathInfo os.FileInfo, err error) error {
		// Stop on error.
		if err != nil {
			return err
		}

		suffix := srcPath[len(src):]
		if strings.HasPrefix(suffix, "/") {
			suffix = suffix[1:]
		}

		// Skip hidden files.
		if strings.HasPrefix(suffix, ".") {
			return nil
		}

		// Skip README and LICENSE.
		if suffix == "LICENSE" || strings.HasPrefix(suffix, "README") {
			return nil
		}

		dstPath := filepath.Join(localConfigDir, suffix)

		// In case we are visiting a directory, create it in the dst.
		if srcPathInfo.IsDir() {
			return os.MkdirAll(dstPath, srcPathInfo.Mode())
		}

		fmt.Println("---> Copy", suffix)

		// Otherwise just copy the file.
		srcFd, err := os.Open(srcPath)
		if err != nil {
			return err
		}
		defer srcFd.Close()

		dstFd, err := os.OpenFile(
			dstPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, srcPathInfo.Mode())
		if err != nil {
			return err
		}
		defer dstFd.Close()

		_, err = io.Copy(dstFd, srcFd)
		return err
	})
}

func cacheDirectoryAbsolutePath() (path string, err error) {
	me, err := user.Current()
	if err != nil {
		return "", nil
	}
	return filepath.Join(me.HomeDir, SkeletonCacheDirname), nil
}
