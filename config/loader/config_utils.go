package loader

import (
	// Stdlib
	"errors"
	"reflect"

	// Internal
	"github.com/salsaflow/salsaflow/config"
)

// ActiveModule returns the active module ID for the given module kind
// as stored in the given local configuration file.
func ActiveModule(local *config.LocalConfig, moduleKind ModuleKind) string {
	var (
		modulesValue = reflect.ValueOf(local).Elem().FieldByName("Modules")
		modulesType  = modulesValue.Type()
		numField     = modulesType.NumField()
		kind         = string(moduleKind)
	)
	for i := 0; i < numField; i++ {
		var (
			fieldValue = modulesValue.Field(i)
			fieldType  = modulesType.Field(i)
		)
		if fieldType.Tag.Get("json") == kind {
			return fieldValue.Interface().(string)
		}
	}

	return ""
}

// SetActiveModule can be used to set the active module ID for the given module kind.
func SetActiveModule(
	local *config.LocalConfig,
	moduleKind ModuleKind,
	moduleId string,
) (modified bool, err error) {

	var (
		modulesValue = reflect.ValueOf(local).Elem().FieldByName("Modules")
		modulesType  = modulesValue.Type()
		numField     = modulesType.NumField()
		kind         = string(moduleKind)
	)
	for i := 0; i < numField; i++ {
		var (
			fieldValue = modulesValue.Field(i)
			fieldType  = modulesType.Field(i)
		)
		if fieldType.Tag.Get("json") != kind {
			continue
		}

		if fieldValue.Interface().(string) == moduleId {
			return false, nil
		}

		fieldValue.SetString(moduleId)
		return true, nil
	}

	return false, errors.New("unknown module kind: " + kind)
}
