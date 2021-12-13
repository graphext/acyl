package pvc

import (
	"fmt"
	"reflect"
)

// SecretStructTag is the default struct tag name
var SecretStructTag = "secret"

// Fill takes a pointer to any struct type and fills any fields annotated with SecretStructTag secret ids.
// Annotated fields must *only* be string or []byte, any other type will cause this method
// to return an error.
// Note that Fill doesn't check the secret type; if the field value is a string, the byte slice returned
// by the backend for that secret will be converted to a string.
func (sc *SecretsClient) Fill(s interface{}) error {
	if s == nil {
		return fmt.Errorf("struct is nil")
	}

	val := reflect.ValueOf(s)
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("s must be a pointer")
	}
	val = val.Elem()
	if !val.IsValid() {
		return fmt.Errorf("s (%#v) is not valid", val)
	}
	v := reflect.Indirect(val)
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("s must be a pointer to a struct")
	}

	for i := 0; i < v.NumField(); i++ {
		// Get the field tag value
		tag := v.Type().Field(i).Tag.Get(SecretStructTag)

		// Skip if tag is not defined or ignored
		if tag == "" || tag == "-" {
			continue
		}

		fn := v.Type().FieldByIndex([]int{i}).Name
		fld := v.Field(i)

		// If we can't set the field, bail out
		if !fld.CanSet() {
			return fmt.Errorf("can't set field %v of type %v", fn, fld.Type().String())
		}

		val, err := sc.Get(tag)
		if err != nil {
			return fmt.Errorf("error getting secret: %v: %w", tag, err)
		}

		// set the field with the secret value
		switch fld.Type().Kind() {
		case reflect.String:
			fld.SetString(string(val))
		case reflect.Slice:
			if fld.Type() == reflect.TypeOf([]byte(nil)) {
				fld.Set(reflect.ValueOf(val))
				break
			}
			// some slice type other than []byte
			fallthrough
		default:
			return fmt.Errorf("unsupported type for field %v: %v", fn, fld.Type().String())
		}
	}
	return nil
}
