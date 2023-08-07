package xs

import (
	"github.com/brianvoe/gofakeit/v6"
	"github.com/getkin/kin-openapi/openapi3"
	"reflect"
	"strings"
)

type ValueResolver func(schema *openapi3.Schema, state *ResolveState) any

type ResolveState struct {
	NamePath                 []string
	ElementIndex             int
	IsHeader                 bool
	ContentType              string
	stopCircularArrayTripOn  int
	stopCircularObjectTripOn string
}

func (s *ResolveState) NewFrom(src *ResolveState) *ResolveState {
	return &ResolveState{
		NamePath:                 src.NamePath,
		IsHeader:                 src.IsHeader,
		ContentType:              src.ContentType,
		stopCircularArrayTripOn:  src.stopCircularArrayTripOn,
		stopCircularObjectTripOn: src.stopCircularObjectTripOn,
	}
}

func (s *ResolveState) WithName(name string) *ResolveState {
	namePath := s.NamePath
	if len(namePath) == 0 {
		namePath = []string{}
	}
	s.stopCircularObjectTripOn = strings.Join(namePath, ".") + "." + name
	namePath = append(namePath, name)

	s.NamePath = namePath
	return s
}

func (s *ResolveState) WithElementIndex(value int) *ResolveState {
	s.stopCircularArrayTripOn = value + 1
	s.ElementIndex = value
	return s
}

func (s *ResolveState) WithHeader() *ResolveState {
	s.IsHeader = true
	return s
}

func (s *ResolveState) WithContentType(value string) *ResolveState {
	s.ContentType = value
	return s
}

func (s *ResolveState) IsCircularObjectTrip() bool {
	return len(s.NamePath) > 0 && s.stopCircularObjectTripOn == strings.Join(s.NamePath, ".")
}

func (s *ResolveState) IsCircularArrayTrip(index int) bool {
	return index+1 == s.stopCircularArrayTripOn
}

func CreateValueResolver() ValueResolver {
	faker := gofakeit.New(0)

	return func(schema *openapi3.Schema, state *ResolveState) any {
		if schema.Example != nil {
			return schema.Example
		}

		switch schema.Type {
		case openapi3.TypeString:
			return faker.Word()
		case openapi3.TypeInteger:
			return faker.Uint32()
		case openapi3.TypeNumber:
			return faker.Uint32()
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
		return IsInteger(value)
	case openapi3.TypeNumber:
		return IsNumber(value)
	case openapi3.TypeBoolean:
		_, ok := value.(bool)
		return ok
	case openapi3.TypeObject:
		return reflect.TypeOf(value).Kind() == reflect.Map
	case openapi3.TypeArray:
		val := reflect.ValueOf(value)
		return val.Kind() == reflect.Slice || val.Kind() == reflect.Array
	default:
		return false
	}
}
