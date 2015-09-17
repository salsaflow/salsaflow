package loader

type ModuleKind string

const (
	ModuleKindIssueTracking ModuleKind = "issue_tracking"
	ModuleKindCodeReview    ModuleKind = "code_review"
	ModuleKindReleaseNotes  ModuleKind = "release_notes"
)

// ConfigSpec represents a complete configuration specification.
// It says how to generate, marshal, unmarshal and validate both local
// and global configuration sections for the given configuration key.
type ConfigSpec interface {

	// ConfigKey returns the globally unique string representing this config spec.
	ConfigKey() string

	// GlobalConfig returns the spec for the global configuration file.
	// The global config is always handled before the local one,
	// so the local spec can access data from the global one without any worries.
	GlobalConfig() ConfigContainer

	// LocalConfig returns the spec for the local configuration file.
	LocalConfig() ConfigContainer
}

// ModuleConfigSpec represents a module config spec, which is a config spec
// that also specified a module kind.
type ModuleConfigSpec interface {
	ConfigSpec

	// ModuleKind returns the module kind for the associated module.
	ModuleKind() ModuleKind
}

// ConfigContainer represents the global or local configuration section
// for the given configuration specification. It specified how to
// generate, marshal, unmarshal and validate the configuration section
// for the key specified by the config spec.
type ConfigContainer interface {

	// PromptUserForConfig is triggered when the config section
	// is not valid or it is not possible to unmarshal it.
	PromptUserForConfig() error
}

// A ConfigContainer can implement Marshaller to overwrite the default
// marshalling mechanism. By default the ConfigContainer is taken as is
// and marshalled (passed to the encoder).
type Marshaller interface {
	Marshal() (interface{}, error)
}

// A ConfigContainer can implement Unmarshaller to overwrite the default
// unmarshalling mechanism. By default the ConfigContainer is just filled
// with the raw data from the associated config section.
type Unmarshaller interface {

	// Unmarshal is passed a function that is to be used to fill
	// an object with the data from the associated config section.
	Unmarshal(unmarshal func(interface{}) error) error
}

// A ConfigContainer can implement Validator to overwrite the default
// validating mechanism. By default the ConfigContainer needs to have
// all exported fields filled to be considered valid.
type Validator interface {
	Validate(sectionPath string) error
}
