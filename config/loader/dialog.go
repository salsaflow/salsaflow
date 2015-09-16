package loader

import (
	// Stdlib
	"os"
	"strconv"
	"text/template"

	// Internal
	"github.com/salsaflow/salsaflow/config"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/prompt"

	// Vendor
	"github.com/fatih/color"
)

// RunCommonBootstrapDialog calls BootstrapConfig for every ConfigSpec
// registered over RegisterBootstrapConfigSpec().
func RunCommonBootstrapDialog() error {
	for _, spec := range bootstrapSpecs {
		if err := BootstrapConfig(spec); err != nil {
			return err
		}
	}
	return nil
}

// ModuleDialogSection represents a section in the module bootstrapping dialog.
type ModuleDialogSection struct {
	AvailableModules []Module
	Optional         bool
}

// RunModuleBootstrapDialog runs the module bootstrapping dialog for every
// section specification passed into the function.
func RunModuleBootstrapDialog(sections ...*ModuleDialogSection) error {
	// Try to read the local configuration file.
	localConfig, err := config.ReadLocalConfig()
	if err != nil {
		if !os.IsNotExist(errs.RootCause(err)) {
			return err
		}
	}

	for _, section := range sections {
		modules := section.AvailableModules

		// In case there are no modules available, skip the section.
		if len(modules) == 0 {
			continue
		}

		// Find the module for the given module kind.
		// Either use the one that is already configured
		// or prompt the user to select one.
		module := tryToFindActiveModule(localConfig, modules)
		if module == nil {
			var err error
			module, err = promptUserToSelectModule(section.AvailableModules, section.Optional)
			if err != nil {
				return err
			}
		}

		// Run the configuration dialog for the selected module.
		// The module can be unset in case it is optional.
		if module != nil {
			if err := BootstrapConfig(module.ConfigSpec()); err != nil {
				return err
			}
		}
	}

	return nil
}

func tryToFindActiveModule(local *config.LocalConfig, modules []Module) Module {
	// In case the local configuration file does not exist,
	// we clearly cannot find the active module.
	if local == nil {
		return nil
	}

	// Get the active module ID.
	kind := modules[0].Kind()
	activeModuleId := ActiveModule(local, kind)

	// Loop over the available modules and try to find matching ID.
	for _, module := range modules {
		if module.Id() == activeModuleId {
			return module
		}
	}
	return nil
}

// Dialog ----------------------------------------------------------------------

var dialogTemplate *template.Template

func init() {
	t := `
Please select a module for module kind '{{.Kind}}':
{{range $index, $element := .Modules}}
  [{{$index | inc}}] {{$element.Id}}{{end}}

Choose a module by inserting the associated number.
Leaving the answer empty will abort the dialog.
{{if .Optional}}
This module kind is optional, insert 's' to skip it.{{end}}
`

	funcs := template.FuncMap{
		"inc": func(i int) int {
			return i + 1
		},
	}

	dialogTemplate = template.Must(template.New("dialog section").Funcs(funcs).Parse(t))
}

type dialogTemplateContext struct {
	Kind     string
	Modules  []Module
	Optional bool
}

func promptUserToSelectModule(modules []Module, optional bool) (Module, error) {
	// Prompt the user to select a module.
	kind := modules[0].Kind()

	for {
		// Write the dialog into the console.
		ctx := &dialogTemplateContext{
			Kind:     string(kind),
			Modules:  modules,
			Optional: optional,
		}
		if err := dialogTemplate.Execute(os.Stdout, ctx); err != nil {
			return nil, err
		}

		// Prompt the user for the answer.
		// An empty answer is aborting the dialog.
		answer, err := prompt.Prompt("You choice: ")
		if err != nil {
			if err == prompt.ErrCanceled {
				prompt.PanicCancel()
			}
			return nil, err
		}

		if optional && answer == "s" {
			color.Cyan("Skipping module kind '%v'", kind)
			return nil, nil
		}

		// Parse the index and return the associated module.
		i, err := strconv.Atoi(answer)
		if err == nil {
			if 0 < i && i <= len(modules) {
				return modules[i-1], nil
			}
		}

		// In case we failed to parse the index or something, run the dialog again.
		color.Yellow("Not a valid choice, please try again!")
	}
}
