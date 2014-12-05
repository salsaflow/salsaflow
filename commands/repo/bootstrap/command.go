package bootstrapCmd

import (
	// Stdlib
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	// Internal
	"github.com/salsaflow/salsaflow/app"
	"github.com/salsaflow/salsaflow/config"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/flag"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/modules"

	// Other
	"gopkg.in/tchap/gocli.v2"
)

var Command = &gocli.Command{
	UsageLine: `
  bootstrap [-issue_tracker=ISSUE_TRACKER]
            [-code_review_tool=CODE_REVIEW_TOOL]
            [-skeleton=SKELETON]`,
	Short: "generate local config for SalsaFlow",
	Long: `
  This command can be used to set up the repository to work with SalsaFlow.

  SalsaFlow needs certain information to be kept in the repository
  to be able to function properly. To make the initial repository setup
  easier, repo bootstrap can be used to generate all necessary files,
  which can be then modified manually if necessary.

  The files are just dumped into the working tree, into .salsaflow directory.
  The directory must be committed after making sure everything is correct.

  Considering the flags, 'issue_tracker' and 'code_review_tool' can be used
  to tell SalsaFlow what implementation to use for particular service modules.
  See the AVAILABLE MODULES section for the allowed values.

  The 'skeleton' flag is a bit different. It can be used to specify
  a GitHub repository that is used as the skeleton for project custom scripts.
  The repository is simply cloned and the contents are poured into .salsaflow.
  The format is '<org>/<repo>', e.g. 'salsaflow/skeleton-golang'.
	`,
	Action: run,
}

const unsetValue = `""`

func init() {
	var (
		issueTrackerKeys   = modules.AvailableIssueTrackerKeys()
		codeReviewToolKeys = modules.AvailableCodeReviewToolKeys()
	)

	// Generate the long description so that it lists the availabe module keys.
	var help bytes.Buffer
	fmt.Fprintln(&help, "AVAILABLE MODULES:\n")
	fmt.Fprintln(&help, "  Issue Trackers")
	fmt.Fprintln(&help, "  --------------")
	fmt.Fprintln(&help, "  These following values can be used for the issue_tracker flag:")
	for _, key := range issueTrackerKeys {
		fmt.Fprintf(&help, "    - %v\n", key)
	}
	fmt.Fprintln(&help)
	fmt.Fprintln(&help, "  Code Review Systems")
	fmt.Fprintln(&help, "  -------------------")
	fmt.Fprintln(&help, "  The following values can be used for the code_review_tool flag:")
	for _, key := range codeReviewToolKeys {
		fmt.Fprintf(&help, "    - %v\n", key)
	}
	Command.Long = fmt.Sprintf("%v\n%v", Command.Long, help.String())

	// Initialise the enum flags.
	flagIssueTracker = flag.NewStringEnumFlag(issueTrackerKeys, unsetValue)
	flagCodeReviewTool = flag.NewStringEnumFlag(codeReviewToolKeys, unsetValue)
}

var (
	flagCodeReviewTool *flag.StringEnumFlag
	flagIssueTracker   *flag.StringEnumFlag
	flagSkeleton       string
)

func init() {
	Command.Flags.Var(flagCodeReviewTool, "code_review_tool",
		"code review tool that is being used for the project")
	Command.Flags.Var(flagIssueTracker, "issue_tracker",
		"issue tracker that is being used for the project")
	Command.Flags.StringVar(&flagSkeleton, "skeleton", flagSkeleton,
		"skeleton to be used to bootstrap the repository")
}

func run(cmd *gocli.Command, args []string) {
	if len(args) != 0 {
		cmd.Usage()
		os.Exit(2)
	}

	// Make sure the required flags are set.
	issueTrackerKey := flagIssueTracker.Value()
	if issueTrackerKey == unsetValue {
		cmd.Usage()
		errs.Fatal(errors.New("flag 'issue_tracker' is not set"))
	}

	codeReviewToolKey := flagCodeReviewTool.Value()
	if codeReviewToolKey == unsetValue {
		cmd.Usage()
		errs.Fatal(errors.New("flag 'code_review_tool' is not set"))
	}

	app.InitOrDie()

	if err := runMain(); err != nil {
		errs.Fatal(err)
	}
}

func runMain() error {
	// Check the global configuration file.
	task := "Check whether the global configuration file exists"
	globalPath, err := config.GlobalConfigFileAbsolutePath()
	if err != nil {
		return errs.NewError(task, err, nil)
	}

	if _, err := os.Stat(globalPath); err != nil {
		if !os.IsNotExist(err) {
			return errs.NewError(task, err, nil)
		}
		log.Warn("Global configuration file not found")
	}

	// Fetch or update the skeleton is necessary.
	if flagSkeleton != "" {
		task := "Fetch or update the given skeleton"
		if err := fetchOrUpdateSkeleton(flagSkeleton); err != nil {
			return errs.NewError(task, err, nil)
		}
	}

	// Make sure the local config directory exists.
	task = "Create the local metadata directory"
	log.Run(task)
	configDir, err := config.LocalConfigDirectoryAbsolutePath()
	if err != nil {
		return errs.NewError(task, err, nil)
	}
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return errs.NewError(task, err, nil)
	}

	// Write the local configuration file.
	task = "Write the local configuration file"
	log.Run(task)
	configPath := filepath.Join(configDir, config.LocalConfigFilename)
	file, err := os.OpenFile(configPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0640)
	if err != nil {
		return errs.NewError(task, err, nil)
	}
	defer file.Close()

	err = WriteLocalConfigTemplate(file, &LocalContext{
		IssueTrackerKey:   flagIssueTracker.Value(),
		CodeReviewToolKey: flagCodeReviewTool.Value(),
	})
	if err != nil {
		return err
	}

	// Move the skeleton files into place.
	if flagSkeleton != "" {
		task := "Copy the skeleton into the metadata directory"
		log.Go(task)
		log.NewLine("")
		if err := pourSkeleton(flagSkeleton, configDir); err != nil {
			return errs.NewError(task, err, nil)
		}
		log.NewLine("")
		log.Ok(task)
	}

	fmt.Println(`
The files were written into .salsaflow directory in the repository root.

Please go through the generated files and make sure they are correct.
Once you are sure they are ok, please commit them into the repository.
`)
	return nil
}
