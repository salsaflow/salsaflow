package config

import (
	// Stdlib
	"fmt"
	"reflect"

	// Internal
	"github.com/salsaflow/salsaflow/log"
)

// EnsureValueFilled returns an error in case the value passed in is not set.
//
// The function checks structs and slices recursively.
func EnsureValueFilled(value interface{}, path string) error {
	logger := log.V(log.Debug)

	// Turn the interface into reflect.Value.
	var (
		v = reflect.ValueOf(value)
		t = v.Type()
	)

	logger.Log(fmt.Sprintf(`config.EnsureValueFilled: Checking "%v" ... `, path))

	// Handle pointers in a special way.
	if kind := v.Kind(); kind == reflect.Ptr || kind == reflect.Slice {
		if v.IsNil() {
			logger.NewLine("  ---> Nil")
			return &ErrKeyNotSet{path}
		}
	}

	// Decide what to do depending on the value kind.
	iv := reflect.Indirect(v)
	switch iv.Kind() {
	case reflect.Struct:
		return ensureStructFilled(iv, path)
	case reflect.Slice:
		return ensureSliceFilled(iv, path)
	}

	// In case the value is not valid, return an error.
	if !v.IsValid() {
		logger.NewLine("  ---> Invalid")
		return &ErrKeyNotSet{path}
	}

	// In case the field is set to the zero value of the given type,
	// we return an error since the field is not set.
	if reflect.DeepEqual(v.Interface(), reflect.Zero(t).Interface()) {
		logger.NewLine("  ---> Unset")
		return &ErrKeyNotSet{path}
	}

	if logger {
		logger.NewLine(fmt.Sprintf("  ---> OK (set to '%v')", v.Interface()))
	}
	return nil
}

func ensureStructFilled(v reflect.Value, path string) error {
	if kind := v.Kind(); kind != reflect.Struct {
		panic(fmt.Errorf("not a struct: %v", kind))
	}

	// Iterate over the struct fields and make sure they are filled,
	// i.e. they are not set to the zero values for their respective types.
	t := v.Type()
	numFields := t.NumField()
	for i := 0; i < numFields; i++ {
		fv := v.Field(i)
		ft := t.Field(i)

		// Skip unexported fields.
		if ft.PkgPath != "" {
			continue
		}

		// Get the field tag.
		tag := ft.Tag.Get("json")
		if tag == "" {
			tag = ft.Name
		}
		fieldPath := fmt.Sprintf("%v.%v", path, tag)
		if err := EnsureValueFilled(fv.Interface(), fieldPath); err != nil {
			return err
		}
	}

	return nil
}

func ensureSliceFilled(v reflect.Value, path string) error {
	if kind := v.Kind(); kind != reflect.Slice {
		panic(fmt.Errorf("not a slice: %v", kind))
	}

	for i := 0; i < v.Len(); i++ {
		if err := EnsureValueFilled(v.Index(i), fmt.Sprintf("%v[%v]", path, i)); err != nil {
			return err
		}
	}
	return nil
}
