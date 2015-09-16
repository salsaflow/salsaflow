package config

import (
	// Stdlib
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	// Internal
	"github.com/salsaflow/salsaflow/errs"
)

func Marshal(v interface{}) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

func Unmarshal(in []byte, v interface{}) error {
	return json.Unmarshal(in, v)
}

// WriteLocalConfig writes the given configuration struct
// into the local configuration file.
//
// In case the target path does not exist, it is created,
// including the parent directories.
//
// In case the file exists, it is truncated.
func writeConfig(absolutePath string, content interface{}, perm os.FileMode) error {
	task := "Write a configuration file"

	// Check the configuration directory and make sure it exists.
	configDir := filepath.Dir(absolutePath)
	info, err := os.Stat(configDir)
	if err != nil {
		if !os.IsNotExist(err) {
			return errs.NewError(task, err)
		}

		// The directory doesn't exist.
		if err := os.MkdirAll(configDir, 0750); err != nil {
			return errs.NewError(task, err)
		}
	}
	if !info.IsDir() {
		return errs.NewError(task, errors.New("not a directory: "+configDir))
	}

	// Marshal the content.
	raw, err := Marshal(content)
	if err != nil {
		return errs.NewError(task, err)
	}

	// Write the raw content into the file.
	if err := ioutil.WriteFile(absolutePath, raw, perm); err != nil {
		return errs.NewError(task, err)
	}
	return nil
}

func readAndUnmarshalConfig(absolutePath string, v interface{}) error {
	// Read the file.
	task := "Read given configuration file"
	content, err := ioutil.ReadFile(absolutePath)
	if err != nil {
		hint := fmt.Sprintf(`
Failed to read the configuration file local at

  %v

`, absolutePath)
		return errs.NewErrorWithHint(task, err, hint)
	}

	// Unmarshall the content.
	task = "Unmarshal given configuration file"
	if err := Unmarshal(content, v); err != nil {
		hint := fmt.Sprintf(`
Failed to parse the configuration file located at

  %v

Make sure the configuration file is valid JSON
that follows the right configuration schema.

`, absolutePath)

		return errs.NewErrorWithHint(task, err, hint)
	}

	return nil
}
