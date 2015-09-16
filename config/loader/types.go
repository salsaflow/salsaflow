package loader

type ModuleKind string

const (
	ModuleKindIssueTracking ModuleKind = "issue_tracking"
	ModuleKindCodeReview    ModuleKind = "code_review"
	ModuleKindReleaseNotes  ModuleKind = "release_notes"
)

type ConfigContainer interface {
	PromptUserForConfig() error
}

type Unmarshaller interface {
	Unmarshal(unmarshal func(interface{}) error) error
}

type Marshaller interface {
	Marshal() (interface{}, error)
}

type Validator interface {
	Validate(sectionPath string) error
}

type ConfigSpec interface {
	ConfigKey() string
	GlobalConfig() ConfigContainer
	LocalConfig() ConfigContainer
}

type ModuleConfigSpec interface {
	ConfigSpec
	ModuleKind() ModuleKind
}
