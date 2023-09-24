package connexions

import (
	"sync"
)

// ReplaceState is a struct that holds information about the current state of the replace operation.
type ReplaceState struct {
	// NamePath is a slice of names of the current element.
	// It is used to build a path to the current element.
	// For example, "users", "name", "first".
	NamePath []string

	// ElementIndex is an index of the current element if required structure to generate is an array.
	ElementIndex int

	// IsHeader is a flag that indicates that the current element we're replacing is a header.
	IsHeader bool

	// IsPathParam is a flag that indicates that the current element we're replacing is a path parameter.
	IsPathParam bool

	// ContentType is a content type of the current element.
	ContentType string

	mu sync.Mutex
}

// NewFrom creates a new ReplaceState instance from the given one.
func (s *ReplaceState) NewFrom(src *ReplaceState) *ReplaceState {
	return &ReplaceState{
		NamePath:    src.NamePath,
		IsHeader:    src.IsHeader,
		IsPathParam: src.IsPathParam,
		ContentType: src.ContentType,
	}
}

func (s *ReplaceState) WithName(name string) *ReplaceState {
	s.mu.Lock()
	defer s.mu.Unlock()

	namePath := s.NamePath
	if len(namePath) == 0 {
		namePath = []string{}
	}
	namePath = append(namePath, name)

	s.NamePath = namePath
	return s
}

func (s *ReplaceState) WithElementIndex(value int) *ReplaceState {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ElementIndex = value
	return s
}

func (s *ReplaceState) WithHeader() *ReplaceState {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.IsHeader = true
	return s
}

func (s *ReplaceState) WithPathParam() *ReplaceState {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.IsPathParam = true
	return s
}

func (s *ReplaceState) WithContentType(value string) *ReplaceState {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ContentType = value
	return s
}
