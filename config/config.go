package config

import (
	"encoding"
	"errors"
	"os"
	"reflect"
	"strconv"
)

func Load(cfg any) error {
	value := reflect.ValueOf(cfg)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return errors.ErrUnsupported
	}

	value = value.Elem()
	return parseStruct(value)
}

func parseStruct(value reflect.Value) error {
	typ := value.Type()
	for i := range typ.NumField() {
		fieldVal := value.Field(i)
		if !fieldVal.CanSet() {
			continue
		}

		field := typ.Field(i)

		def := field.Tag.Get("default")
		if env := field.Tag.Get("env"); env != "" {
			def = os.Getenv(env)
		}

		if unmarshaler, ok := fieldVal.Addr().Interface().(encoding.TextUnmarshaler); ok {
			if err := unmarshaler.UnmarshalText([]byte(def)); err != nil {
				return err
			}
		}

		fieldTyp := field.Type
		if fieldTyp.Kind() == reflect.Struct {
			if err := parseStruct(fieldVal); err != nil {
				return err
			}
			continue
		}

		if def == "" {
			continue
		}

		if err := parseValue(fieldVal, def); err != nil {
			return err
		}
	}

	return nil
}

func parseValue(value reflect.Value, literal string) error {
	typ := value.Type()
	switch typ.Kind() {
	case reflect.Bool:
		b, err := strconv.ParseBool(literal)
		if err != nil {
			return err
		}
		value.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(literal, 0, typ.Bits())
		if err != nil {
			return err
		}
		value.SetInt(i)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u, err := strconv.ParseUint(literal, 0, typ.Bits())
		if err != nil {
			return err
		}
		value.SetUint(u)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(literal, typ.Bits())
		if err != nil {
			return err
		}
		value.SetFloat(f)
	case reflect.String:
		value.SetString(literal)
	}

	return nil
}
