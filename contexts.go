package xs

import (
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/rawbytes"
	"strings"
)

type ParsedContextResult struct {
	Result  map[string]any
	Aliases map[string]string
}

func ParseContextFile(filePath string) (*ParsedContextResult, error) {
	k := koanf.New(".")
	provider := file.Provider(filePath)
	if err := k.Load(provider, yaml.Parser()); err != nil {
		return nil, err
	}

	return parseContext(k)
}

func ParseContextFromBytes(content []byte) (*ParsedContextResult, error) {
	k := koanf.New(".")
	provider := rawbytes.Provider(content)
	if err := k.Load(provider, yaml.Parser()); err != nil {
		return nil, err
	}

	return parseContext(k)
}

func parseContext(k *koanf.Koanf) (*ParsedContextResult, error) {
	fakes := GetFakes()

	transformed := koanf.New(".")
	aliased := make(map[string]string)
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
			} else if strings.HasPrefix(v, "func:") {

				// } else if strings.HasPrefix(v, "env:") {
				// 	envName := v[4:]
				// 	if envValue := GetEnv(envName); envValue != "" {
				// 		value = envValue
				// 	}
			} else if strings.HasPrefix(v, "alias:") {
				alias := v[6:]
				aliased[key] = alias
			} else if strings.HasPrefix(v, "botify:") {

			} else if strings.HasPrefix(v, "from-path:") {

			}
		}
		_ = transformed.Set(key, value)
	}

	// for key, required := range aliased {
	// 	if aliasValue := transformed.Get(required); aliasValue != nil {
	// 		_ = transformed.Set(key, aliasValue)
	// 	}
	// }

	target := make(map[string]any)
	if err := transformed.Unmarshal("", &target); err != nil {
		return nil, err
	}

	return &ParsedContextResult{
		Result:  target,
		Aliases: aliased,
	}, nil
}

func CollectContexts(names []map[string]string, available map[string]map[string]any) []map[string]any {
	res := make([]map[string]any, 0)

	for _, contextProps := range names {
		for key, value := range contextProps {
			if ctx, exists := available[key]; exists {
				// child key passed. there's no need to pass complete context
				if value != "" {
					if subCtx, subExists := ctx[value]; subExists {
						if subCtxMap, ok := subCtx.(map[string]any); ok {
							ctx = subCtxMap
						}
					}
				}
				res = append(res, ctx)
			}
		}
	}
	return res
}
