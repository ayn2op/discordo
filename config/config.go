package config

import (
	"errors"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

func Load(cfg any) error {
	value := reflect.ValueOf(cfg)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return errors.ErrUnsupported
	}

	value = value.Elem()
	if value.Kind() != reflect.Struct {
		return errors.ErrUnsupported
	}

	return parseStruct(value)
}

func parseStruct(value reflect.Value) error {
	typ := value.Type()

	for i := range typ.NumField() {
		fieldValue := value.Field(i)
		if !fieldValue.CanSet() {
			continue
		}

		field := typ.Field(i)
		fieldTyp := field.Type
		def := field.Tag.Get("default")
		env := field.Tag.Get("env")
		if env != "" {
			def = os.Getenv(env)
		}

		if unmarshaler, ok := fieldValue.Addr().Interface().(toml.Unmarshaler); ok {
			if def != "" {
				if err := unmarshaler.UnmarshalTOML(def); err != nil {
					return err
				}
			}
		}

		if fieldTyp.Kind() == reflect.Struct {
			if err := parseStruct(fieldValue); err != nil {
				return err
			}
			continue
		}

		if def == "" {
			continue
		}

		if err := parseValue(fieldValue, def); err != nil {
			return err
		}
	}

	return nil
}

func parseValue(value reflect.Value, def string) error {
	typ := value.Type()
	kind := typ.Kind()

	switch kind {

	case reflect.Bool:
		b, err := strconv.ParseBool(def)
		if err != nil {
			return err
		}
		value.SetBool(b)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(def, 0, typ.Bits())
		if err != nil {
			return err
		}
		value.SetInt(i)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u, err := strconv.ParseUint(def, 0, typ.Bits())
		if err != nil {
			return err
		}
		value.SetUint(u)

	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(def, typ.Bits())
		if err != nil {
			return err
		}
		value.SetFloat(f)

	case reflect.String:
		value.SetString(def)

	case reflect.Slice, reflect.Array:
		parts := strings.Fields(def)
		if kind == reflect.Array && len(parts) != typ.Len() {
			return errors.New("invalid array length")
		}

		if kind == reflect.Slice {
			value.Set(reflect.MakeSlice(typ, len(parts), len(parts)))
		}

		for i, part := range parts {
			elem := value.Index(i)
			if err := parseValue(elem, part); err != nil {
				return err
			}
		}
	default:
		return errors.ErrUnsupported
	}

	return nil
}
