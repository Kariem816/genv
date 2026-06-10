package genv

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/joho/godotenv"
)

type TransformerFn func(varName string, varStrVal string) (any, error)
type varMode int

const (
	modeBoth varMode = iota
	modeDevOnly
	modeProdOnly
)

type configVar struct {
	Key         string
	mode        varMode
	Transformer TransformerFn
}

var transformers = map[string]TransformerFn{}

func setFieldByName(obj any, fieldName string, value any) error {
	v := reflect.ValueOf(obj)

	if v.Kind() != reflect.Pointer || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("expected a pointer to a struct")
	}

	v = v.Elem()
	field := v.FieldByName(fieldName)

	if !field.IsValid() {
		return fmt.Errorf("no such field: %s in struct", fieldName)
	}

	if !field.CanSet() {
		return fmt.Errorf("cannot set field: %s", fieldName)
	}

	field.Set(reflect.ValueOf(value).Convert(field.Type()))

	return nil
}

func readConfig(f reflect.StructField) (configVar, error) {
	fName := f.Name
	tag, ok := f.Tag.Lookup("env")
	if !ok {
		return configVar{}, fmt.Errorf("GENV: field `%s` is not configured", fName)
	}

	pieces := strings.Split(tag, ";")
	if len(pieces) < 2 || len(pieces) > 3 {
		return configVar{}, fmt.Errorf("GENV: field `%s` config should match the format `<Key>;<d|p|b>[;transformer]`", fName)
	}

	cv := configVar{}

	key := strings.TrimSpace(pieces[0])
	if len(key) == 0 {
		return configVar{}, fmt.Errorf("GENV: field `%s` config should specify a key", fName)
	}
	cv.Key = key

	scope := strings.TrimSpace(pieces[1])
	if len(scope) != 1 {
		return configVar{}, fmt.Errorf("GENV: field `%s` scope should either be d|p|b (dev, prod, both)", fName)
	}
	switch scope {
	case "d":
		cv.mode = modeDevOnly
	case "p":
		cv.mode = modeProdOnly
	case "b": // implecitly both
	default:
		return configVar{}, fmt.Errorf("GENV: field `%s` mode should either be d|p|b (dev, prod, both)", fName)
	}

	if len(pieces) == 2 {
		return cv, nil
	}

	t := strings.TrimSpace(pieces[2])
	if len(t) == 0 {
		return configVar{}, fmt.Errorf("GENV: field `%s` transformer can't be empty", fName)
	}

	ter, ok := transformers[t]
	if !ok {
		return configVar{}, fmt.Errorf("GENV: field `%s` transformer (%s) is not registered", fName, t)
	}

	cv.Transformer = ter

	return cv, nil
}

func validateConfig(env tEnv, name string, cv configVar) (any, bool, error) {
	val, ok := os.LookupEnv(cv.Key)
	if !ok {
		if (env == envDev && cv.mode == modeDevOnly) || (env == envProd && cv.mode == modeProdOnly) {
			return nil, false, fmt.Errorf("GENV: `%s` variable is required", name)
		}
		return "", false, nil
	}

	if cv.Transformer != nil {
		vot, err := cv.Transformer(name, val)
		return vot, true, err
	}

	return val, true, nil
}

var env tEnv

func init() {
	err := godotenv.Load()
	if err != nil {
		fmt.Printf("[Warning] GENV: couldn't load .env file. %v\n", err)
	}

	envVal, ok := os.LookupEnv("GENV")
	if !ok {
		fmt.Println("[Warning] GENV: `GENV` variable is not set. Set it to either dev|prod.")
		return
	}

	env.Scan(envVal)
}

func IsDev() bool {
	return env == envDev
}

func IsProd() bool {
	return env == envProd
}

func collectFields(typ reflect.Type) []reflect.StructField {
	fields := []reflect.StructField{}

	for f := range typ.Fields() {
		if f.Anonymous && f.Type.Kind() == reflect.Struct {
			fmt.Printf("[Info] GENV: collecting fields from embedded struct `%s` in struct `%s`\n", f.Type.Name(), typ.Name())
			fields = append(fields, collectFields(f.Type)...)
		} else {
			fields = append(fields, f)
		}
	}

	return fields
}

func Parse[T any](Cfg *T) error {
	typ := reflect.TypeFor[T]()
	if typ.Kind() != reflect.Struct {
		return fmt.Errorf("GENV: expected a struct type but got %s", typ.Kind())
	}

	fields := collectFields(typ)
	for _, f := range fields {
		name := f.Name

		cv, err := readConfig(f)
		if err != nil {
			return err
		}

		val, ok, err := validateConfig(env, f.Name, cv)
		if err != nil {
			return err
		}
		if cv.Transformer != nil {
			if f.Type != reflect.TypeOf(val) {
				return fmt.Errorf("GENV: field `%s` transformer returned type %T, expected %s", name, val, f.Type.Name())
			}
		} else {
			if f.Type.Kind() != reflect.String {
				return fmt.Errorf("GENV: field `%s` is of type %s, expected string or a transformer", name, f.Type.Name())
			}
		}

		if ok { // some variables might not be needed in some environments
			err = setFieldByName(Cfg, name, val)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func RegisterTransformer(name string, fn TransformerFn) {
	transformers[name] = fn
}
