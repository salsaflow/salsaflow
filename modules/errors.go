package modules

import (
	// Stdlib
	"fmt"

	// Internal
	"github.com/salsaflow/salsaflow/config/loader"
)

// ErrModuleNotFound is returned when the module with the specified ID
// cannot be found in the list of registered modules.
type ErrModuleNotFound struct {
	moduleId string
}

func (err *ErrModuleNotFound) Error() string {
	return fmt.Sprintf("module '%v' not found", err.moduleId)
}

// ErrModuleNotSet is returned when the module of the given kind is not active,
// in other words, no module ID is associated with the given module kind.
type ErrModuleNotSet struct {
	moduleKind loader.ModuleKind
}

func (err *ErrModuleNotSet) Error() string {
	return fmt.Sprintf("no module registered for module kind '%v'", err.moduleKind)
}

// ErrInvalidModule is returned when the module cannot be cast
// to the right module kind interface.
type ErrInvalidModule struct {
	moduleId   string
	moduleKind loader.ModuleKind
}

func (err *ErrInvalidModule) Error() string {
	return fmt.Sprintf(
		"module '%v' is not a valid module of kind '%v'", err.moduleId, err.moduleKind)
}
