package xs

import (
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/rawbytes"
	"strings"
)

func ParseContextFile(filePath string) (map[string]any, error) {
	k := koanf.New(".")
	provider := file.Provider(filePath)
	if err := k.Load(provider, yaml.Parser()); err != nil {
		return nil, err
	}

	return parseContext(k)
}

func ParseContextFromBytes(content []byte) (map[string]any, error) {
	k := koanf.New(".")
	provider := rawbytes.Provider(content)
	if err := k.Load(provider, yaml.Parser()); err != nil {
		return nil, err
	}

	return parseContext(k)
}

func parseContext(k *koanf.Koanf) (map[string]any, error) {
	fakes := GetFakes()

	transformed := koanf.New(".")
	for key, value := range k.All() {
		if v, isString := value.(string); isString {
			if strings.HasPrefix(v, "fake:") {
				funcName := v[5:]
				// function name can be explicitly set or inferred from key
				if funcName == "" {
					funcName = key
				}
				if fn, exists := fakes[funcName]; exists {
					value = fn
				}
			} else if strings.HasPrefix(v, "alias:") {
				alias := v[6:]
				if aliasValue := transformed.Get(alias); aliasValue != nil {
					value = aliasValue
				}
			} else if strings.HasPrefix(v, "botify:") {

			}
		}
		_ = transformed.Set(key, value)
	}

	target := make(map[string]any)
	if err := transformed.Unmarshal("", &target); err != nil {
		return nil, err
	}

	return target, nil
}

func ReplaceMapFunctionPlaceholders(data any, funcs map[string]any) any {
	switch value := data.(type) {
	case map[string]any:
		newMap := make(map[string]any)
		for key, val := range value {
			newMap[key] = ReplaceMapFunctionPlaceholders(val, funcs)
		}
		return newMap
	case string:
		if strings.HasPrefix(value, "func:") {
			funcName := value[5:]
			if fn, exists := funcs[funcName]; exists {
				return fn
			}
		}
		return value
	default:
		return value
	}
}
