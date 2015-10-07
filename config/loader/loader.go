package loader

import (
	// Stdlib
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	// Internal
	"github.com/salsaflow/salsaflow/config"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/log"

	// Vendor
	"github.com/fatih/color"
)

// BootstrapConfig can be used to bootstrap SalsaFlow configuration
// according to the given configuration specification.
func BootstrapConfig(spec ConfigSpec) error {
	if err := bootstrapGlobalConfig(spec); err != nil {
		return err
	}
	return bootstrapLocalConfig(spec)
}

// LoadConfig can be used to load SalsaFlow configuration
// according to the given configuration specification.
//
// The main difference between BootstrapConfig and LoadConfig is that
// LoadConfig returns an error when the local configuration is not valid.
// While `repo bootstrap` is using BootstrapConfig, all other modules
// and commands should be using LoadConfig. The local configuration file
// is only supposed to be generated once during `repo bootstrap`.
func LoadConfig(spec ConfigSpec) (err error) {
	if err := bootstrapGlobalConfig(spec); err != nil {
		return err
	}
	return loadLocalConfig(spec)
}

func bootstrapGlobalConfig(spec ConfigSpec) error {
	// Run the common loading function with the right arguments.
	task := "Bootstrap global config according to the spec"
	if err := load(&loadArgs{
		configKind:      "global",
		configKey:       spec.ConfigKey(),
		configContainer: spec.GlobalConfig(),
		readConfig:      readGlobalConfig,
		emptyConfig:     emptyGlobalConfig,
	}); err != nil {
		return errs.NewError(task, err)
	}
	return nil
}

func bootstrapLocalConfig(spec ConfigSpec) error {
	task := "Bootstrap local configuration according to the spec"

	// Some handy variables.
	configKey := spec.ConfigKey()
	container := spec.LocalConfig()
	moduleSpec, isModuleSpec := spec.(ModuleConfigSpec)

	// Pre-read local config.
	// It is needed the pre-write hook as well.
	local, err := config.ReadLocalConfig()
	readConfig := func() (configFile, error) {
		return local, err
	}

	// Handle module config specs a bit differently.
	var preWriteHook func() (bool, error)
	if isModuleSpec {
		// Make sure the config container is not nil in case this is a module config.
		// In case the local config container is nil, the pre-write hook is not executed
		// and the active module ID is not set, and that would be a problem.
		// Returning a nil local config container is a valid choice, but we still
		// need the pre-write hook to be executed to set the active module ID.
		if container == nil {
			container = newEmptyModuleConfigContainer(configKey, moduleSpec.ModuleKind())
		}

		// In case this is a module config spec, set the the pre-write hook
		// to modify the local config file to activate the module being configured.
		preWriteHook = func() (bool, error) {
			return SetActiveModule(local, moduleSpec.ModuleKind(), configKey)
		}
	}

	// The post-write hook simply tells the user to commit the local config file.
	postWriteHook := func() error {
		fmt.Println()
		log.Warn("Local configuration file modified, please commit it.")
		return nil
	}

	// Run the common loading function with the right arguments.
	if err := load(&loadArgs{
		configKind:      "local",
		configKey:       configKey,
		configContainer: container,
		readConfig:      readConfig,
		emptyConfig:     emptyLocalConfig,
		preWriteHook:    preWriteHook,
		postWriteHook:   postWriteHook,
	}); err != nil {
		return errs.NewError(task, err)
	}
	return nil
}

func loadLocalConfig(spec ConfigSpec) error {
	// Run the common loading function with the right arguments.
	// Setting disallowPrompt to true makes the function return an error
	// in case the config record is not valid instead of running the dialog.
	task := "Load local configuration according to the spec"
	if err := load(&loadArgs{
		configKind:      "local",
		configKey:       spec.ConfigKey(),
		configContainer: spec.LocalConfig(),
		readConfig:      readLocalConfig,
		disallowPrompt:  true,
	}); err != nil {
		return errs.NewError(task, err)
	}
	return nil
}

type configFile interface {
	ConfigRecord(configKey string) (*config.ConfigRecord, error)
	SetConfigRecord(configKey string, rawConfig []byte)
	SaveChanges() error
}

type loadArgs struct {
	configKind      string
	configKey       string
	configContainer ConfigContainer
	readConfig      func() (configFile, error)
	emptyConfig     func() configFile
	preWriteHook    func() (bool, error)
	postWriteHook   func() error
	disallowPrompt  bool
}

func load(args *loadArgs) error {
	// Save the args into regular variables.
	var (
		configKind     = args.configKind
		configKey      = args.configKey
		container      = args.configContainer
		readConfig     = args.readConfig
		emptyConfig    = args.emptyConfig
		disallowPrompt = args.disallowPrompt
	)

	// Do nothing in case the container is nil.
	if container == nil {
		return nil
	}

	// Read the configuration file.
	configFile, err := readConfig()
	if err != nil {
		if emptyConfig == nil || !os.IsNotExist(errs.RootCause(err)) {
			return err
		}
		// In case the file is not there, initialise a new one.
		configFile = emptyConfig()
	}

	prompt := func(err error) error {
		if disallowPrompt {
			err := errors.New("configuration dialog disabled")
			task := "Prompt the user for configuration according to the spec"
			hint := `
The configuration dialog for the local configuration file can only be run
during 'repo bootstrap' so that the configuration file is not modified
without anybody noticing in the middle of other work being done.

Please fix the issues manually, either by manually editing the local
configuration file or by re-running 'repo bootstrap' command.

Don't forget to commit the changes.

`
			return errs.NewErrorWithHint(task, err, hint)
		}
		return promptAndWrite(configFile, args)
	}

	// Find the config record for the given key.
	// In case there is an error and the prompt is allowed, prompt the user.
	section, err := configFile.ConfigRecord(configKey)
	if err != nil {
		return prompt(err)
	}

	// Unmarshal the record according to the spec.
	// In case there is an error and the prompt is allowed, prompt the user.
	if err := unmarshal(section.RawConfig, container); err != nil {
		fmt.Println()
		log.Log(fmt.Sprintf(
			"Failed to unmarshal %v configuration, will try to run the bootstrap dialog",
			configKind))
		log.NewLine(fmt.Sprintf("(err = %v)", err.Error()))
		return prompt(err)
	}

	// Validate the returned object according to the spec.
	// In case there is an error and the prompt is allowed, prompt the user.
	if err := validate(container, section.Path()); err != nil {
		fmt.Println()
		log.Log(fmt.Sprintf(
			"%v configuration section invalid, will try to run the bootstrap dialog",
			strings.Title(configKind)))
		log.NewLine(fmt.Sprintf("(error = %v)", err.Error()))
		return prompt(err)
	}

	return nil
}

func promptAndWrite(configFile configFile, args *loadArgs) (err error) {
	// Save the args into regular variables.
	var (
		configKind    = args.configKind
		configKey     = args.configKey
		container     = args.configContainer
		preWriteHook  = args.preWriteHook
		postWriteHook = args.postWriteHook
	)

	// Tell the user what is happening.
	fmt.Println()
	color.Cyan("-----> Configuring '%v' (%v configuration)", configKey, configKind)
	fmt.Println()

	defer func() {
		// In case we are panicking, forward the panic.
		if r := recover(); r != nil {
			panic(r)
		}

		// Otherwise print the colored message.
		fmt.Println()
		if err == nil {
			color.Green("-----> Done")
		} else {
			color.Red("-----> Error")
		}
	}()

	// Prompt the user according to the spec.
	if err := container.PromptUserForConfig(); err != nil {
		return err
	}

	// Marshal the result.
	raw, err := marshal(container)
	if err != nil {
		return err
	}

	// Store the result in the config file object.
	var modified bool
	if raw != nil {
		configFile.SetConfigRecord(configKey, json.RawMessage(raw))
		modified = true
	}

	// Optionally run the pre-write hook.
	if preWriteHook != nil {
		changedConfig, err := preWriteHook()
		if err != nil {
			return err
		}
		modified = modified || changedConfig
	}

	// In case the config file hasn't been modified, we can return.
	if !modified {
		return nil
	}

	// Write the config file.
	if err := configFile.SaveChanges(); err != nil {
		return err
	}

	// Optionally run the post-write hook.
	if postWriteHook != nil {
		if err := postWriteHook(); err != nil {
			return err
		}
	}

	return nil
}

// unmarshal unmarshals data into container.
//
// In case the container implements Unmarshaller interface,
// the custom unmarshalling function is used. Otherwise data
// is simply unmarshalled into container.
func unmarshal(data []byte, container ConfigContainer) error {
	unmarshalFunc := func(v interface{}) error {
		return config.Unmarshal(data, v)
	}

	var err error
	unmarshaller, ok := container.(Unmarshaller)
	if ok {
		err = unmarshaller.Unmarshal(unmarshalFunc)
	} else {
		err = unmarshalFunc(container)
	}
	return err
}

// marshal marshals container and returns the raw data.
//
// In case the container implements Marshaller interface,
// the custom marshalling function is used. Otherwise
// the container itself is simply marshalled directly.
func marshal(container ConfigContainer) ([]byte, error) {
	var (
		v   interface{}
		err error
	)
	marshaller, ok := container.(Marshaller)
	if ok {
		v, err = marshaller.Marshal()
	} else {
		v = container
	}
	if err != nil {
		return nil, err
	}

	return config.Marshal(v)
}

// validate validates the configuration stored in the container.
//
// In case the container implements Validator interface,
// the custom validation function is used. Otherwise config.EnsureValueFilled
// is used, which means that all exported fields must be set for the config
// to be treated as valid.
func validate(container ConfigContainer, sectionPath string) error {
	var err error
	validator, ok := container.(Validator)
	if ok {
		err = validator.Validate(sectionPath)
	} else {
		err = config.EnsureValueFilled(container, sectionPath)
	}
	return err
}

func readLocalConfig() (configFile, error) {
	return config.ReadLocalConfig()
}

func emptyLocalConfig() configFile {
	return config.NewEmptyLocalConfig()
}

func readGlobalConfig() (configFile, error) {
	return config.ReadGlobalConfig()
}

func emptyGlobalConfig() configFile {
	return config.NewEmptyGlobalConfig()
}
