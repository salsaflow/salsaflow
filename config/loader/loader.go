package loader

import (
	// Stdlib
	"encoding/json"
	"fmt"
	"os"

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
	task := "Bootstrap local config according to the spec"

	// Pre-read local config since it is needed
	// in the pre-write hook as well.
	local, err := config.ReadLocalConfig()
	readConfig := func() (configFile, error) {
		return local, err
	}

	// In case the spec is a ModuleConfigSpec, this pre-write hook
	// modifies the local config file to activate the module being configured.
	configKey := spec.ConfigKey()
	preWriteHook := func() (bool, error) {
		moduleSpec, ok := spec.(ModuleConfigSpec)
		if ok {
			return SetActiveModule(local, moduleSpec.ModuleKind(), configKey)
		}
		return false, nil
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
		configContainer: spec.LocalConfig(),
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
	task := "Load local config according to the spec"
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
		configKey      = args.configKey
		container      = args.configContainer
		readConfig     = args.readConfig
		emptyConfig    = args.emptyConfig
		disallowPrompt = args.disallowPrompt
	)

	// Do nothing in case the container is nil.
	// That means that the config is not specified at all.
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
			return err
		}
		return promptAndWrite(configFile, args)
	}

	// Find the config record for the given key.
	section, err := configFile.ConfigRecord(configKey)
	if err != nil {
		return prompt(err)
	}

	// Unmarshal the record according to the spec.
	// In case there is an error and the prompt is allowed, prompt the user.
	if err := unmarshal(section.RawConfig, container); err != nil {
		return prompt(err)
	}

	// Validate the returned object according to the spec.
	// In case there is an error and the prompt is allowed, prompt the user.
	if err := validate(container, section.Path()); err != nil {
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
	if err != nil {
		fmt.Println()
		log.Log("Failed to unmarshal configuration, will run the bootstrap dialog")
		log.NewLine(fmt.Sprintf("(err = %v)", err.Error()))
		return err
	}
	return nil
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
	// In case v is nil, return nil as well.
	// This means that no config needs to be stored.
	if v == nil {
		return nil, nil
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
	if err != nil {
		fmt.Println()
		log.Log("Configuration section invalid, will run the bootstrap dialog")
		log.NewLine(fmt.Sprintf("(error = %v)", err.Error()))
		return err
	}
	return nil
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
