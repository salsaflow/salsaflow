package repo

import (
	// Stdlib
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	// Internal
	"github.com/salsaflow/salsaflow/config"
	"github.com/salsaflow/salsaflow/prompt"

	// Vendor
	"github.com/fatih/color"
	"github.com/shiena/ansicolor"
)

type Config interface {
	SalsaFlowEnabledTimestamp() time.Time
}

type LocalConfig struct {
	EnabledTimestamp time.Time `yaml:"salsaflow_enabled_timestamp"`
}

func (config *LocalConfig) SalsaFlowEnabledTimestamp() time.Time {
	return config.EnabledTimestamp
}

var cache *LocalConfig

func LoadConfig() (Config, error) {
	if cache == nil {
		var lc LocalConfig
		if err := config.UnmarshalLocalConfig(&lc); err != nil {
			return nil, err
		}
		if lc.EnabledTimestamp.IsZero() {
			// The enabled timestamp must be set no matter what.
			printSalsaFlowEnabledTimestampWarning()
			return nil, errors.New("SalsaFlow enabled timestamp not set")
		}
		cache = &lc
	}
	return cache, nil
}

func printSalsaFlowEnabledTimestampWarning() (n int64, err error) {
	// Open the console to make sure the user can always see it.
	stdout, err := prompt.OpenConsole(os.O_WRONLY)
	if err != nil {
		return 0, err
	}
	defer stdout.Close()

	// Generate the warning.
	var output bytes.Buffer

	redBold := color.New(color.FgRed).Add(color.Bold).SprintFunc()
	fmt.Fprintln(&output, redBold("\nWarning: 'salsaflow_enabled_timestamp' key missing."))

	red := color.New(color.FgRed).SprintFunc()
	fmt.Fprintln(&output, red("Please set the key in the local configuration file."))
	fmt.Fprintln(&output, red("The format is: 2014-09-02T12:36:11.142902641+01:00\n"))

	// Dump it into the console.
	return io.Copy(ansicolor.NewAnsiColorWriter(stdout), &output)
}
