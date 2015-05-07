package hooks

import (
	// Stdlib
	"bytes"
	"fmt"
	"io"
	"time"

	// Internal
	"github.com/salsaflow/salsaflow/config"
	"github.com/salsaflow/salsaflow/errs"

	// Vendor
	"github.com/fatih/color"
	"github.com/shiena/ansicolor"
)

type LocalConfig struct {
	EnabledTimestamp time.Time `yaml:"salsaflow_enabled_timestamp"`
}

func SalsaFlowEnabledTimestamp() (time.Time, error) {
	task := "Load hook-related SalsaFlow config"

	var lc LocalConfig
	if err := config.UnmarshalLocalConfig(&lc); err != nil {
		return time.Time{}, errs.NewError(task, err, nil)
	}

	return lc.EnabledTimestamp, nil
}

func PrintSalsaFlowEnabledTimestampWarning(writer io.Writer) (n int64, err error) {
	var output bytes.Buffer

	redBold := color.New(color.FgRed).Add(color.Bold).SprintFunc()
	fmt.Fprintln(&output, redBold("Warning: 'salsaflow_enabled_timestamp' key missing."))

	red := color.New(color.FgRed).SprintFunc()
	fmt.Fprintln(&output, red("Please set the key in the local configuration file."))
	fmt.Fprintln(&output, red("The format is: 2014-09-02T12:36:11.142902641+01:00"))

	return io.Copy(ansicolor.NewAnsiColorWriter(writer), &output)
}
