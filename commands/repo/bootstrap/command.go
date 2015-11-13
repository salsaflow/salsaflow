package bootstrapCmd

import (
	// Stdlib
	"errors"
	"fmt"
	"os"

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
  bootstrap -skeleton=SKELETON [-skeleton_only]

  bootstrap -no_skeleton`,
	Short: "bootstrap repository for SalsaFlow",
	Long: `
  Bootstrap the repository for SalsaFlow.

  This command should be used to set up the local configuration directory
  for SalsaFlow (the directory that is then committed into the repository).

  The user is prompted for all necessary data.

  The -skeleton flag can be used to specify the repository to be used
  for custom scripts. It expects a string of "$OWNER/$REPO" and then uses
  the repository located at github.com/$OWNER/$REPO. It clones the repository
  and copies the scripts directory into the local configuration directory.

  In case no skeleton is to be used to bootstrap the repository,
  -no_skeleton must be specified explicitly.

  In case the repository is bootstrapped, but the skeleton is missing,
  it can be added by specifying -skeleton=SKELETON -skeleton_only.
  That will skip the configuration file generation step.
	`,
	Action: run,
}

var (
	flagNoSkeleton   bool
	flagSkeleton     string
	flagSkeletonOnly bool
)

func init() {
	// Register flags.
	Command.Flags.BoolVar(&flagNoSkeleton, "no_skeleton", flagNoSkeleton,
		"do not use any skeleton to bootstrap the repository")
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
	switch {
	case flagSkeleton == "" && !flagNoSkeleton:
		cmd.Usage()
		return errs.NewError(
			task, errors.New("-no_skeleton must be specified when no skeleton is given"))

	case flagSkeletonOnly && flagSkeleton == "":
		cmd.Usage()
		return errs.NewError(
			task, errors.New("-skeleton must be specified when -skeleton_only is set"))
	}

	// Make sure the local config directory exists.
	act, err := ensureLocalConfigDirectoryExists()
	if err != nil {
		return err
	}
	defer action.RollbackOnError(&err, act)

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

func ensureLocalConfigDirectoryExists() (action.Action, error) {
	task := "Make sure the local configuration directory exists"

	// Get the directory absolute path.
	localConfigDir, err := config.LocalConfigDirectoryAbsolutePath()
	if err != nil {
		return nil, errs.NewError(task, err)
	}

	// In case the path exists, make sure it is a directory.
	info, err := os.Stat(localConfigDir)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, errs.NewError(task, err)
		}
	} else {
		if !info.IsDir() {
			return nil, errs.NewError(task, fmt.Errorf("not a directory: %v", localConfigDir))
		}
		return action.Noop, nil
	}

	// Otherwise create the directory.
	if err := os.MkdirAll(localConfigDir, 0755); err != nil {
		return nil, errs.NewError(task, err)
	}

	// Return the rollback function.
	act := action.ActionFunc(func() error {
		// Delete the directory.
		log.Rollback(task)
		task := "Delete the local configuration directory"
		if err := os.RemoveAll(localConfigDir); err != nil {
			return errs.NewError(task, err)
		}
		return nil
	})
	return act, nil
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
