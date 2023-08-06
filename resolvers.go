package xs

import (
	"github.com/brianvoe/gofakeit/v6"
	"github.com/getkin/kin-openapi/openapi3"
	"reflect"
)

type ValueResolver func(schema *openapi3.Schema, state *ResolveState) any

type ResolveState struct {
	NamePath    []string
	Example     any
	IsHeader    bool
	ContentType string
}

func (s *ResolveState) addPath(name string) *ResolveState {
	namePath := s.NamePath
	if len(namePath) == 0 {
		namePath = []string{}
	}

	return &ResolveState{
		NamePath:    append(namePath, name),
		Example:     s.Example,
		IsHeader:    s.IsHeader,
		ContentType: s.ContentType,
	}
}

func (s *ResolveState) markAsHeader() *ResolveState {
	return &ResolveState{
		NamePath:    s.NamePath,
		Example:     s.Example,
		IsHeader:    true,
		ContentType: s.ContentType,
	}
}

func (s *ResolveState) setContentType(value string) *ResolveState {
	return &ResolveState{
		NamePath:    s.NamePath,
		Example:     s.Example,
		IsHeader:    s.IsHeader,
		ContentType: value,
	}
}

func CreateValueResolver() ValueResolver {
	faker := gofakeit.New(0)

	return func(schema *openapi3.Schema, state *ResolveState) any {
		namePath := state.NamePath
		for _, name := range namePath {
			if name == "id" {
				return faker.Uint32()
			} else if name == "first" {
				return faker.Person().FirstName
			} else if name == "last" {
				return faker.Person().LastName
			} else if name == "age" {
				return 21
			} else if name == "name" {
				return faker.PetName()
			} else if name == "tag" {
				return faker.Gamertag()
			}
		}

		if state.Example != nil {
			return state.Example
		}

		switch schema.Type {
		case openapi3.TypeString:
			return faker.Word()
		case openapi3.TypeInteger:
			return faker.Uint32()
		case openapi3.TypeNumber:
			return faker.Float32()
		case openapi3.TypeBoolean:
			return faker.Bool()
		}

		return nil
	}
}

func IsCorrectlyResolvedType(value any, needed string) bool {
	switch needed {
	case openapi3.TypeString:
		_, ok := value.(string)
		return ok
	case openapi3.TypeInteger:
		_, ok := value.(int)
		return ok
	case openapi3.TypeNumber:
		_, ok := value.(float32)
		return ok
	case openapi3.TypeBoolean:
		_, ok := value.(bool)
		return ok
	case openapi3.TypeObject:
		return reflect.TypeOf(value).Kind() == reflect.Map
	case openapi3.TypeArray:
		_, ok := value.([]interface{})
		return ok
	default:
		return false
	}
}
