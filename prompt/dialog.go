package prompt

import (
	// Stdlib
	"fmt"
	"reflect"
	"strconv"

	// Vendor
	"github.com/bgentry/speakeasy"
)

func Dialog(value interface{}, questionPrefix string) error {
	fmt.Println("Just press Enter to use the default value (if available).")
	fmt.Println()
	return dialogStruct(value, questionPrefix)
}

func dialogStruct(value interface{}, questionPrefix string) error {
	var (
		v = reflect.Indirect(reflect.ValueOf(value))
		t = v.Type()
	)
	if kind := v.Kind(); kind != reflect.Struct {
		panic(fmt.Errorf("not a struct: %v", kind))
	}

	numFields := t.NumField()
	for i := 0; i < numFields; i++ {
		fv := v.Field(i)
		ft := t.Field(i)

		// Skip unexported fields.
		if ft.PkgPath != "" {
			continue
		}

		// Fill structs recursively.
		if kind := reflect.Indirect(fv).Kind(); kind == reflect.Struct {
			if err := dialogStruct(fv.Addr().Interface(), questionPrefix); err != nil {
				return err
			}
			continue
		}

		var (
			questionSuffix = ft.Tag.Get("prompt")
			defaultValue   = ft.Tag.Get("default")
			secret         = ft.Tag.Get("secret") != ""
		)
		if questionSuffix == "" {
			continue
		}

		question := fmt.Sprintf("%v %v", questionPrefix, questionSuffix)
		var (
			answer string
			err    error
		)
		if secret {
			answer, err = speakeasy.Ask(question + ": ")
			if err == nil && answer == "" {
				err = ErrCanceled
			}
		} else {
			if defaultValue != "" {
				answer, err = PromptDefault(question, defaultValue)
			} else {
				answer, err = Prompt(question + ": ")
			}
		}
		if err != nil {
			if err == ErrCanceled {
				PanicCancel()
			}
			return err
		}

		switch fv.Kind() {
		case reflect.Int:
			i, err := strconv.Atoi(answer)
			if err != nil {
				return err
			}
			fv.SetInt(int64(i))

		case reflect.String:
			fv.SetString(answer)

		default:
			return fmt.Errorf("unsupported field type: %v", fv.Kind())
		}
	}
	return nil
}
