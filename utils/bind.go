package utils

import (
	"errors"
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func StructToMap(item interface{}) map[string]interface{} {
	out := make(map[string]interface{})

	v := reflect.ValueOf(item)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// we only accept structs
	if v.Kind() != reflect.Struct {
		log.Fatal("ToMap only accepts structs; got %T", v)
	}

	typ := v.Type()

	for i := 0; i < v.NumField(); i++ {
		// gets us a StructField
		fi := typ.Field(i)
		tagv := fi.Tag.Get("redis")

		if tagv == "" || tagv == "-" {
			continue
		}

		val := v.Field(i).Interface()

		if fi.Type == reflect.TypeOf(time.Now()) {
			val = val.(time.Time).Unix()
		}

		out[tagv] = val
	}

	return out
}

func Bind(data map[string]string, ptr interface{}) error {
	if ptr == nil || len(data) == 0 {
		return nil
	}

	typ := reflect.TypeOf(ptr).Elem()
	val := reflect.ValueOf(ptr).Elem()

	// Map
	if typ.Kind() == reflect.Map {
		for k, v := range data {
			val.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v[0]))
		}

		return nil
	}

	// !struct
	if typ.Kind() != reflect.Struct {
		return errors.New("binding element must be a struct")
	}

	for i := 0; i < typ.NumField(); i++ {
		typeField := typ.Field(i)
		structField := val.Field(i)

		if !structField.CanSet() {
			continue
		}

		//structFieldKind := structField.Kind()
		inputFieldName := typeField.Tag.Get("redis")

		if inputFieldName == "" || inputFieldName == "-" {
			continue
		}

		inputValue, exists := data[inputFieldName]

		if !exists {
			// Go json.Unmarshal supports case insensitive binding.  However the
			// url params are bound case sensitive which is inconsistent.  To
			// fix this we must check all of the map values in a
			// case-insensitive search.
			for k, v := range data {
				if strings.EqualFold(k, inputFieldName) {
					inputValue = v
					exists = true
					break
				}
			}
		}

		if !exists {
			continue
		}

		if err := setWithProperType(typeField.Type.Kind(), inputValue, structField); err != nil {
			return err
		}
	}

	return nil
}

func setWithProperType(valueKind reflect.Kind, val string, structField reflect.Value) error {
	switch valueKind {
	case reflect.Ptr:
		return setWithProperType(structField.Elem().Kind(), val, structField.Elem())
	case reflect.Int:
		return setIntField(val, 0, structField)
	case reflect.Int8:
		return setIntField(val, 8, structField)
	case reflect.Int16:
		return setIntField(val, 16, structField)
	case reflect.Int32:
		return setIntField(val, 32, structField)
	case reflect.Int64:
		return setIntField(val, 64, structField)
	case reflect.Uint:
		return setUintField(val, 0, structField)
	case reflect.Uint8:
		return setUintField(val, 8, structField)
	case reflect.Uint16:
		return setUintField(val, 16, structField)
	case reflect.Uint32:
		return setUintField(val, 32, structField)
	case reflect.Uint64:
		return setUintField(val, 64, structField)
	case reflect.Bool:
		return setBoolField(val, structField)
	case reflect.Float32:
		return setFloatField(val, 32, structField)
	case reflect.Float64:
		return setFloatField(val, 64, structField)
	case reflect.String:
		structField.SetString(val)
	case reflect.Struct:
		switch structField.Type() {
		case reflect.TypeOf(time.Now()):
			timeInt, _ := strconv.Atoi(val)
			timeVal := time.Unix(int64(timeInt), 0)
			structField.Set(reflect.ValueOf(timeVal))
		default:
			return errors.New("unknown type")
		}
	default:
		return errors.New("unknown type")
	}

	return nil
}

func setIntField(value string, bitSize int, field reflect.Value) error {
	if value == "" {
		value = "0"
	}

	intVal, err := strconv.ParseInt(value, 10, bitSize)

	if err == nil {
		field.SetInt(intVal)
	}

	return err
}

func setUintField(value string, bitSize int, field reflect.Value) error {
	if value == "" {
		value = "0"
	}

	uintVal, err := strconv.ParseUint(value, 10, bitSize)

	if err == nil {
		field.SetUint(uintVal)
	}

	return err
}

func setBoolField(value string, field reflect.Value) error {
	if value == "" {
		value = "false"
	}

	boolVal, err := strconv.ParseBool(value)

	if err == nil {
		field.SetBool(boolVal)
	}

	return err
}

func setFloatField(value string, bitSize int, field reflect.Value) error {
	if value == "" {
		value = "0.0"
	}

	floatVal, err := strconv.ParseFloat(value, bitSize)

	if err == nil {
		field.SetFloat(floatVal)
	}

	return err
}
