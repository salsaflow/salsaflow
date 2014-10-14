// +build darwin linux

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

	// Unlink the existing executable.
	if exists {
		if err := os.Remove(dstPath); err != nil {
			return err
		}
	}

	// Write the new executable.
	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0755)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}
