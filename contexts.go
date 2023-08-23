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

func CollectContexts(names []map[string]string, fileCollections map[string]map[string]any,
	initial map[string]any) []map[string]any {
	res := make([]map[string]any, 0)

	if len(initial) > 0 {
		res = append(res, initial)
	}

	for _, contextProps := range names {
		for key, value := range contextProps {
			if ctx, exists := fileCollections[key]; exists {
				// name := key
				// child key passed. there's no need to pass complete context
				if value != "" {
					if subCtx, subExists := ctx[value]; subExists {
						// name = value
						if subCtxMap, ok := subCtx.(map[string]any); ok {
							ctx = subCtxMap
						}
					}
				}
				// log.Printf("context %s added.", name)
				res = append(res, ctx)
			}
		}
	}
	return res
}

func parseContext(k *koanf.Koanf) (*ParsedContextResult, error) {
	fakes := GetFakes()
	oneArgFuncs := GetFakeFuncFactoryWithString()

	transformed := koanf.New(".")
	aliased := make(map[string]string)
	for key, value := range k.All() {
		v, isString := value.(string)
		if !isString {
			_ = transformed.Set(key, value)
			continue
		}

		parts := strings.Split(v, ":")
		prefix := strings.ToLower(parts[0])
		var res any
		var parsed bool

		switch prefix {
		case "fake":
			res, parsed = parseFakeContextFunc(key, parts, fakes)
		case "func":
			res, parsed = parseOneArgContextFunc(parts, oneArgFuncs)
		case "alias":
			alias := v[6:]
			aliased[key] = alias
		case "botify":
			res, parsed = parseBotifyContextFunc(parts, oneArgFuncs)
		}

		if parsed {
			value = res
		}
		_ = transformed.Set(key, value)
	}

	target := make(map[string]any)
	if err := transformed.Unmarshal("", &target); err != nil {
		return nil, err
	}

	return &ParsedContextResult{
		Result:  target,
		Aliases: aliased,
	}, nil
}

func parseFakeContextFunc(key string, valueParts []string, available map[string]FakeFunc) (FakeFunc, bool) {
	if len(valueParts) < 2 {
		return nil, false
	}
	value := valueParts[1]

	// function name can be explicitly set or inferred from key
	if value == "" {
		value = key
	}
	if fn, exists := available[value]; exists {
		return fn, true
	}
	return nil, false
}

func parseOneArgContextFunc(valueParts []string, available map[string]FakeFuncFactoryWithString) (FakeFunc, bool) {
	if len(valueParts) < 3 {
		return nil, false
	}

	funcName := valueParts[1]
	if fn, exists := available[funcName]; exists {
		arg1 := valueParts[2]
		// call the factory function with the argument
		return fn(arg1), true
	}
	return nil, false
}

func parseBotifyContextFunc(valueParts []string, available map[string]FakeFuncFactoryWithString) (FakeFunc, bool) {
	if len(valueParts) < 2 {
		return nil, false
	}
	pattern := valueParts[1]

	if pattern == "" {
		return nil, false
	}
	if fn, exists := available["botify"]; exists {
		return fn(pattern), true
	}
	return nil, false
}
