package config

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/knadh/koanf/maps"
	"github.com/knadh/koanf/parsers/dotenv"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type parseOption struct {
	envPrefix  string
	delimiter  string
	tag        string
	flatStruct bool
	helpFlag   string

	configPathKey string
	configFile    bool

	descriptionTag string
	flagTag        string
	shortTag       string
}

type ParseOption func(*parseOption)

func WithEnvPrefix(prefix string) ParseOption {
	return func(po *parseOption) {
		po.envPrefix = prefix
	}
}

func WithDelimiter(delimiter string) ParseOption {
	return func(po *parseOption) {
		po.delimiter = delimiter
	}
}

func WithStructTagName(tag string) ParseOption {
	return func(po *parseOption) {
		po.tag = tag
	}
}

func WithDescriptionStructTagName(tag string) ParseOption {
	return func(po *parseOption) {
		po.descriptionTag = tag
	}
}

// WithoutConfigFile removes the -c or --config flag from the command
func WithoutConfigFile() ParseOption {
	return func(po *parseOption) {
		po.configFile = false
	}
}

type Validatable interface {
	Validate() error
}

// Parse takes every object and is able to fill and validate that object depending on config file, env file and flag values.
// https://github.com/knadh/koanf
// Your passed struct must define . delimited koanf struct tags in order to match env/.env and flag values to your struct.
// Additionally your struct may define a Validate() error method which is called at the end of parsing the config
// Registers flags and returns a parser function that can be used as PreRunE.
func RegisterFlags[T any](config *T, persistent bool, app *cobra.Command, options ...ParseOption) func() error {

	op := parseOption{
		envPrefix:  "PREFIX_",
		delimiter:  ".",
		tag:        "koanf",
		flatStruct: true,
		helpFlag:   "help",

		configPathKey: "config",
		configFile:    true,

		descriptionTag: "description",
		flagTag:        "flag",
		shortTag:       "short",
	}

	for _, o := range options {
		o(&op)
	}

	envToKoanf := func(s string) string {
		return strings.ToLower(strings.ReplaceAll(s, "_", op.delimiter))
	}

	f := func(s string) string {
		return envToKoanf(strings.TrimPrefix(s, op.envPrefix))
	}

	koanfToEnv := func(s string) string {
		return op.envPrefix + strings.ToUpper(strings.ReplaceAll(s, op.delimiter, "_"))
	}

	defaults := koanf.New(op.delimiter)

	// does not error
	_ = defaults.Load(structs.ProviderWithDelim(config, op.tag, op.delimiter), nil)

	var fs *pflag.FlagSet
	if persistent {
		fs = app.PersistentFlags()
	} else {
		fs = app.Flags()
	}

	if op.configFile {
		fs.StringP(op.configPathKey, "c", "", fmt.Sprintf(".env config file path (or via env variable %s%s)", op.envPrefix, strings.ToUpper(op.configPathKey)))
	}

	ct := reflect.TypeOf(config)
	if ct.Kind() == reflect.Pointer {
		ct = ct.Elem()
	}

	defaultMap, _ := maps.Flatten(defaults.All(), nil, op.delimiter)
	maxKeyLen := maxKeyLen(defaultMap)
	padding := maxKeyLen + len(op.envPrefix) + 1
	format := "\n  %-" + strconv.Itoa(padding) + "s   %s"
	var sb strings.Builder
	sb.Grow((padding + 6) * len(defaultMap) * 3)

	// register flags for all known struct fields

	if ct.NumField() > 0 {
		sb.WriteString("Environment variables:")
	}

	for i := 0; i < ct.NumField(); i++ {
		field := ct.Field(i)
		sTag := field.Tag
		key, found := sTag.Lookup(op.tag)
		if !found || key == "-" {
			continue
		}
		v := defaultMap[key]

		desc := sTag.Get(op.descriptionTag)
		short := sTag.Get(op.shortTag)
		flag := sTag.Get(op.flagTag)

		envName := koanfToEnv(key)
		// key, description
		sb.WriteString(fmt.Sprintf(format, envName, desc))

		// allow skipping of flag creation for specific fields
		if flag == "false" {
			continue
		}

		// key is now a flag name
		flagName := strings.ReplaceAll(key, op.delimiter, "-")

		if v != nil {
			// default value if not empty
			defaultVal := fmt.Sprintf("%v", v)
			if defaultVal != "" {
				sb.WriteString(fmt.Sprintf(" (default: %q)", defaultVal))
			}
		}

		switch x := v.(type) {
		case bool:
			if len(short) == 1 {
				fs.BoolP(flagName, short, x, desc)
			} else {
				fs.Bool(flagName, x, desc)
			}
		case int8:
			if len(short) == 1 {
				fs.Int8P(flagName, short, x, desc)
			} else {
				fs.Int8(flagName, x, desc)
			}
		case int16:
			if len(short) == 1 {
				fs.Int16P(flagName, short, x, desc)
			} else {
				fs.Int16(flagName, x, desc)
			}
		case int32:
			if len(short) == 1 {
				fs.Int32P(flagName, short, x, desc)
			} else {
				fs.Int32(flagName, x, desc)
			}
		case int:
			if len(short) == 1 {
				fs.IntP(flagName, short, x, desc)
			} else {
				fs.Int(flagName, x, desc)
			}
		case int64:
			if len(short) == 1 {
				fs.Int64P(flagName, short, x, desc)
			} else {
				fs.Int64(flagName, x, desc)
			}
		case uint8:
			if len(short) == 1 {
				fs.Uint8P(flagName, short, x, desc)
			} else {
				fs.Uint8(flagName, x, desc)
			}
		case uint16:
			if len(short) == 1 {
				fs.Uint16P(flagName, short, x, desc)
			} else {
				fs.Uint16(flagName, x, desc)
			}
		case uint32:
			if len(short) == 1 {
				fs.Uint32P(flagName, short, x, desc)
			} else {
				fs.Uint32(flagName, x, desc)
			}
		case uint:
			if len(short) == 1 {
				fs.UintP(flagName, short, x, desc)
			} else {
				fs.Uint(flagName, x, desc)
			}
		case uint64:
			if len(short) == 1 {
				fs.Uint64P(flagName, short, x, desc)
			} else {
				fs.Uint64(flagName, x, desc)
			}
		case float64:
			if len(short) == 1 {
				fs.Float64P(flagName, short, x, desc)
			} else {
				fs.Float64(flagName, x, desc)
			}
		case float32:
			if len(short) == 1 {
				fs.Float32P(flagName, short, x, desc)
			} else {
				fs.Float32(flagName, x, desc)
			}
		case time.Duration:
			if len(short) == 1 {
				fs.DurationP(flagName, short, x, desc)
			} else {
				fs.Duration(flagName, x, desc)
			}

		default:
			strValue := ""
			if v != nil {
				strValue = fmt.Sprintf("%v", v)
			}

			if len(short) == 1 {
				fs.StringP(flagName, short, strValue, desc)
			} else {
				fs.String(flagName, strValue, desc)
			}
		}
	}

	sb.WriteString("\n")

	app.Long += sb.String()

	return func() error {

		environment := koanf.New(op.delimiter)
		err := environment.Load(env.Provider(op.envPrefix, op.delimiter, f), nil)
		if err != nil {
			return err
		}

		// disable unknonwn flags errors
		before := fs.ParseErrorsWhitelist
		fs.ParseErrorsWhitelist.UnknownFlags = true
		defer func() {
			fs.ParseErrorsWhitelist = before
		}()

		err = fs.Parse(os.Args)
		if err != nil {
			return fmt.Errorf("failed to parse config flags: %w", err)
		}

		flagSet := koanf.New(op.delimiter)
		err = flagSet.Load(
			posflag.ProviderWithValue(
				fs,
				op.delimiter,
				nil,
				func(key, value string) (string, interface{}) {
					return envToKoanf(key), value
				},
			), nil)
		if err != nil {
			return err
		}

		// skip parsing of the config in case we encounter the help flag
		if flagSet.Bool(op.helpFlag) {
			return nil
		}

		dotenvFile := koanf.New(op.delimiter)

		if op.configFile {
			// flags found -> use flags
			// flags not found -> env found -> use env
			// if someone defined a default location, try that
			for _, k := range []*koanf.Koanf{flagSet, environment, defaults} {
				p := k.Get(op.configPathKey)
				if p == nil {
					continue
				}
				switch configPath := p.(type) {
				case string:
					// we only want to access strings, not any sub maps
					if configPath == "" {
						continue
					}
					err = dotenvFile.Load(file.Provider(configPath), dotenv.ParserEnv(op.envPrefix, op.delimiter, f))
					if err != nil {
						return err
					}
				default:
					continue
				}
				break
			}

		}

		k := koanf.New(op.delimiter)
		err = k.Merge(defaults)
		if err != nil {
			return err
		}

		// only support .env files format
		err = k.Merge(dotenvFile)
		if err != nil {
			return err
		}

		err = k.Merge(environment)
		if err != nil {
			return err
		}

		// merge flag map into struct map
		_ = k.Load(confmap.Provider(flagSet.All(), "-"), nil)

		err = k.UnmarshalWithConf("", config, koanf.UnmarshalConf{
			FlatPaths: op.flatStruct,
		})
		if err != nil {
			return err
		}

		var a any = config
		if v, ok := a.(Validatable); ok {
			return v.Validate()
		}
		return nil
	}
}

func maxKeyLen(m map[string]any) int {
	maxLen := 1
	for k := range m {
		keyLen := len(k)
		if keyLen > maxLen {
			maxLen = keyLen
		}
	}

	return maxLen
}

type dotEnvParseOption struct {
	envPrefix string
	delimiter string
	tag       string
	flatPaths bool
}

func MarshalDotEnv(cfgs ...any) ([]byte, error) {
	op := dotEnvParseOption{
		envPrefix: "PREFIX_",
		delimiter: ".",
		tag:       "koanf",
		flatPaths: true,
	}

	k := koanf.New(op.delimiter)

	for _, cfg := range cfgs {
		err := k.Load(structs.ProviderWithDelim(cfg, op.tag, op.delimiter), nil)
		if err != nil {
			return nil, err
		}
	}

	koanfToEnv := func(s string) string {
		return op.envPrefix + strings.ToUpper(strings.ReplaceAll(s, op.delimiter, "_"))
	}

	m, _ := maps.Flatten(k.All(), nil, op.delimiter)

	for key, value := range m {
		k.Delete(key)
		err := k.Set(koanfToEnv(key), value)
		if err != nil {
			return nil, fmt.Errorf("failed to set key %s: %w", key, err)
		}
	}

	dotEnv := dotenv.ParserEnv(op.envPrefix, op.delimiter, func(s string) string { return s })
	return k.Marshal(dotEnv)
}
