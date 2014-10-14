package pkg

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func replaceExecutable(src io.Reader, dstDir, dstBase string) error {
	dstPath := filepath.Join(dstDir, dstBase)

	// Check the destination path.
	var exists bool
	info, err := os.Stat(dstPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	} else {
		exists = true
		if !info.Mode().IsRegular() {
			return fmt.Errorf("not a regular file: %v", dstPath)
		}
	}

	// Make sure the old file is deleted.
	renameDstPath := filepath.Join(dstDir, fmt.Sprintf("_%v.old", dstBase))

	info, err = os.Stat(renameDstPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	} else {
		if err := os.Remove(renameDstPath); err != nil {
			return err
		}
	}

	// Rename the existing executable.
	if exists {
		if err := os.Rename(dstPath, renameDstPath); err != nil {
			return err
		}
	}

	// Write the new executable.
	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0640)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}
