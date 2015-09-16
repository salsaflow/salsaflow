package bootstrapCmd

import (
	// Stdlib
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"

	// Internal
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/app"
	"github.com/salsaflow/salsaflow/app/appflags"
	"github.com/salsaflow/salsaflow/config"
	"github.com/salsaflow/salsaflow/config/loader"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/modules"
	"github.com/salsaflow/salsaflow/prompt"

	// Other
	"gopkg.in/tchap/gocli.v2"
)

var Command = &gocli.Command{
	UsageLine: `
  bootstrap [-skeleton=SKELETON] [-skeleton_only]`,
	Short: "bootstrap repository for SalsaFlow",
	Long: `
	`,
	Action: run,
}

var (
	flagSkeleton     string
	flagSkeletonOnly bool
)

func init() {
	// Register flags.
	Command.Flags.StringVar(&flagSkeleton, "skeleton", flagSkeleton,
		"skeleton to be used to bootstrap the repository")
	Command.Flags.BoolVar(&flagSkeletonOnly, "skeleton_only", flagSkeletonOnly,
		"skip the config dialog and only install the skeleton")

	// Register global flags.
	appflags.RegisterGlobalFlags(&Command.Flags)
}

func run(cmd *gocli.Command, args []string) {
	if len(args) != 0 {
		cmd.Usage()
		os.Exit(2)
	}

	app.InitLogging()

	defer prompt.RecoverCancel()

	if err := runMain(cmd); err != nil {
		errs.Fatal(err)
	}
}

func runMain(cmd *gocli.Command) (err error) {
	// Validate CL flags.
	task := "Check the command line flags"
	if flagSkeletonOnly && flagSkeleton == "" {
		cmd.Usage()
		return errs.NewError(
			task, errors.New("-skeleton must be specified when -skeleton_only is set"))
	}

	// Make sure the local config directory exists.
	if err := ensureLocalConfigDirectoryExists(); err != nil {
		return err
	}
	action.RollbackOnError(&err, action.ActionFunc(deleteLocalConfigDirectory))

	// Set up the global and local configuration file unless -skeleton_only.
	if !flagSkeletonOnly {
		if err := assembleAndWriteConfig(); err != nil {
			return err
		}
	}

	// Install the skeleton into the local config directory if desired.
	if skeleton := flagSkeleton; skeleton != "" {
		if err := getAndPourSkeleton(skeleton); err != nil {
			return err
		}
	}

	fmt.Println()
	log.Log("Successfully bootstrapped the repository for SalsaFlow")
	log.NewLine("Do not forget to commit modified configuration files!")
	return nil
}

func ensureLocalConfigDirectoryExists() error {
	task := "Make sure the local configuration directory exists"

	// Get the directory absolute path.
	localConfigDir, err := config.LocalConfigDirectoryAbsolutePath()
	if err != nil {
		return errs.NewError(task, err)
	}

	// In case the path exists, make sure it is a directory.
	info, err := os.Stat(localConfigDir)
	if err != nil {
		if !os.IsNotExist(err) {
			return errs.NewError(task, err)
		}
	} else {
		if !info.IsDir() {
			return errs.NewError(task, fmt.Errorf("not a directory: %v", localConfigDir))
		}
		return nil
	}

	// Otherwise create the directory.
	if err := os.MkdirAll(localConfigDir, 0755); err != nil {
		return errs.NewError(task, err)
	}

	return nil
}

func deleteLocalConfigDirectory() error {
	task := "Delete the local configuration directory"
	log.Run(task)

	// Get the directory absolute path.
	localConfigDir, err := config.LocalConfigDirectoryAbsolutePath()
	if err != nil {
		return errs.NewError(task, err)
	}

	// Delete the directory.
	if err := os.RemoveAll(localConfigDir); err != nil {
		return errs.NewError(task, err)
	}

	return nil
}

func assembleAndWriteConfig() error {
	// Group available modules by kind.
	var (
		issueTrackingModules []loader.Module
		codeReviewModules    []loader.Module
		releaseNotesModules  []loader.Module
	)
	groups := groupModulesByKind(modules.AvailableModules())
	for _, group := range groups {
		switch group[0].Kind() {
		case loader.ModuleKindIssueTracking:
			issueTrackingModules = group
		case loader.ModuleKindCodeReview:
			codeReviewModules = group
		case loader.ModuleKindReleaseNotes:
			releaseNotesModules = group
		}
	}

	sort.Sort(commonModules(issueTrackingModules))
	sort.Sort(commonModules(codeReviewModules))
	sort.Sort(commonModules(releaseNotesModules))

	// Run the common dialog.
	task := "Run the core configuration dialog"
	if err := loader.RunCommonBootstrapDialog(); err != nil {
		return errs.NewError(task, err)
	}

	// Run the dialog.
	task = "Run the modules configuration dialog"
	err := loader.RunModuleBootstrapDialog(
		&loader.ModuleDialogSection{issueTrackingModules, false},
		&loader.ModuleDialogSection{codeReviewModules, false},
		&loader.ModuleDialogSection{releaseNotesModules, true},
	)
	if err != nil {
		return errs.NewError(task, err)
	}

	return nil
}

func getAndPourSkeleton(skeleton string) error {
	// Get or update given skeleton.
	task := fmt.Sprintf("Get or update skeleton '%v'", skeleton)
	log.Run(task)
	if err := getOrUpdateSkeleton(flagSkeleton); err != nil {
		return errs.NewError(task, err)
	}

	// Move the skeleton files into place.
	task = "Copy the skeleton into the configuration directory"
	log.Go(task)

	localConfigDir, err := config.LocalConfigDirectoryAbsolutePath()
	if err != nil {
		return errs.NewError(task, err)
	}

	log.NewLine("")
	if err := pourSkeleton(flagSkeleton, localConfigDir); err != nil {
		return errs.NewError(task, err)
	}
	log.NewLine("")
	log.Ok(task)

	return nil
}

func writeConfigFile(path string, configObject interface{}) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0640)
	if err != nil {
		return err
	}
	defer file.Close()

	content, err := config.Marshal(configObject)
	if err != nil {
		return err
	}

	if _, err := io.Copy(file, bytes.NewReader(content)); err != nil {
		return err
	}

	return nil
}
