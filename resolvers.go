package xs

import (
	"github.com/brianvoe/gofakeit/v6"
	"github.com/getkin/kin-openapi/openapi3"
	"reflect"
	"sync"
)

type ValueResolver func(schema *openapi3.Schema, state *ResolveState) any

type ResolveState struct {
	NamePath           []string
	IsHeader           bool
	ContentType        string
	CircularArrayTrip  int
	CircularObjectTrip string
	mu                 sync.Mutex
}

func (s *ResolveState) Copy(src *ResolveState) *ResolveState {
	return &ResolveState{
		NamePath:           src.NamePath,
		IsHeader:           src.IsHeader,
		ContentType:        src.ContentType,
		CircularArrayTrip:  src.CircularArrayTrip,
		CircularObjectTrip: src.CircularObjectTrip,
		mu:                 sync.Mutex{},
	}
}

func (s *ResolveState) WithName(name string) *ResolveState {
	s.mu.Lock()
	defer s.mu.Unlock()
	namePath := s.NamePath
	if len(namePath) == 0 {
		namePath = []string{}
	}
	namePath = append(namePath, name)

	s.NamePath = namePath
	// s.CircularObjectTrip = namePath
	return s
}

func (s *ResolveState) WithHeader() *ResolveState {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.IsHeader = true
	return s
}

func (s *ResolveState) WithContentType(value string) *ResolveState {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ContentType = value
	return s
}

func (s *ResolveState) WithCircularArrayTrip(value int) *ResolveState {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.CircularArrayTrip = value
	return s
}

func (s *ResolveState) WithCircularObjectTrip(trip string) *ResolveState {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.CircularObjectTrip = trip
	return s
}

// func (s *ResolveState) IsCircularObjectTrip() bool {
// 	s.mu.Lock()
// 	defer s.mu.Unlock()
// 	return len(s.NamePath) > 0 && IsSlicesEqual[string](s.NamePath, s.CircularObjectTrip)
// }

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

		if schema.Example != nil {
			return schema.Example
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
