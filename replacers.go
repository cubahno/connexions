package xs

import (
	"github.com/brianvoe/gofakeit/v6"
	"github.com/getkin/kin-openapi/openapi3"
	"log"
	"reflect"
	"strings"
	"sync"
)

type ValueReplacer func(schemaOrContent any, state *ReplaceState) any

type ReplaceState struct {
	NamePath                 []string
	ElementIndex             int
	IsHeader                 bool
	IsURLParam               bool
	ContentType              string
	stopCircularArrayTripOn  int
	stopCircularObjectTripOn string
	mu                       sync.Mutex
}

func (s *ReplaceState) NewFrom(src *ReplaceState) *ReplaceState {
	return &ReplaceState{
		NamePath:                 src.NamePath,
		IsHeader:                 src.IsHeader,
		ContentType:              src.ContentType,
		stopCircularArrayTripOn:  src.stopCircularArrayTripOn,
		stopCircularObjectTripOn: src.stopCircularObjectTripOn,
	}
}

func (s *ReplaceState) WithName(name string) *ReplaceState {
	s.mu.Lock()
	defer s.mu.Unlock()

	namePath := s.NamePath
	if len(namePath) == 0 {
		namePath = []string{}
	}
	s.stopCircularObjectTripOn = strings.Join(namePath, ".") + "." + name
	namePath = append(namePath, name)

	s.NamePath = namePath
	return s
}

func (s *ReplaceState) WithElementIndex(value int) *ReplaceState {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopCircularArrayTripOn = value + 1
	s.ElementIndex = value
	return s
}

func (s *ReplaceState) WithHeader() *ReplaceState {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.IsHeader = true
	return s
}

func (s *ReplaceState) WithURLParam() *ReplaceState {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.IsURLParam = true
	return s
}

func (s *ReplaceState) WithContentType(value string) *ReplaceState {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ContentType = value
	return s
}

func (s *ReplaceState) IsCircularObjectTrip() bool {
	return len(s.NamePath) > 0 && s.stopCircularObjectTripOn == strings.Join(s.NamePath, ".")
}

func (s *ReplaceState) IsCircularArrayTrip(index int) bool {
	return index+1 == s.stopCircularArrayTripOn
}

func CreateValueSchemaReplacer() ValueReplacer {
	faker := gofakeit.New(0)

	return func(content any, state *ReplaceState) any {
		schema, ok := content.(*openapi3.Schema)
		if !ok {
			log.Printf("content is not *openapi3.Schema, but %s", reflect.TypeOf(content))
			return nil
		}

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

func CreateValueContentReplacer() ValueReplacer {
	faker := gofakeit.New(0)

	return func(content any, state *ReplaceState) any {
		switch content.(type) {
		case string:
			if state.IsURLParam {
				return faker.Uint8()
			}
		}
		return content
	}
}

func IsCorrectlyReplacedType(value any, needed string) bool {
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
