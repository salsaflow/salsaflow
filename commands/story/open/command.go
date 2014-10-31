package openCmd

import (
	// Stdlib
	"fmt"
	"os"

	// Internal
	"github.com/salsita/salsaflow/app"
	"github.com/salsita/salsaflow/errs"
	"github.com/salsita/salsaflow/log"
	"github.com/salsita/salsaflow/modules"

	// Other
	"gopkg.in/tchap/gocli.v1"
)

var Command = &gocli.Command{
	UsageLine: "open STORY",
	Short:     "open the given story in the web browser",
	Long: `
  Open the issue tracker in the web browser at the home page of the given story.
	`,
	Action: run,
}

func run(cmd *gocli.Command, args []string) {
	if len(args) != 1 {
		cmd.Usage()
		os.Exit(2)
	}

	app.MustInit()

	if err := runMain(args[0]); err != nil {
		errs.Fatal(err)
	}
}

func runMain(storyId string) error {
	task := fmt.Sprintf("Open story %s", storyId)
	log.Run(task)
	if err := modules.GetIssueTracker().OpenStory(storyId); err != nil {
		return errs.NewError(task, err, nil)
	}
	return nil
}
