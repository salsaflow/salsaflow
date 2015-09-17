package loader

type Module interface {

	// Id returns the string globally identifying the module.
	// Module authors are advised to use a reversed URL as seen in Java world,
	// e.g. com.example.codereview.servicename
	Id() string

	// Kind returns the kind of this module.
	Kind() ModuleKind

	// ConfigSpec returns the configuration specification
	// to be used when bootstrapping the module.
	ConfigSpec() ModuleConfigSpec
}
