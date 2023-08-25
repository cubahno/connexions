package connexions

import (
	"sync"
)

type ReplaceState struct {
	NamePath     []string
	ElementIndex int
	IsHeader     bool
	IsPathParam  bool
	ContentType  string
	refPath      []string
	mu           sync.Mutex
}

func (s *ReplaceState) NewFrom(src *ReplaceState) *ReplaceState {
	return &ReplaceState{
		NamePath:    src.NamePath,
		IsHeader:    src.IsHeader,
		ContentType: src.ContentType,
		refPath:     src.refPath,
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

func (s *ReplaceState) WithReference(name string) *ReplaceState {
	s.mu.Lock()
	defer s.mu.Unlock()

	if name == "" {
		return s
	}

	refPath := s.refPath
	if len(refPath) == 0 {
		refPath = make([]string, 0)
	}
	refPath = append(refPath, name)
	s.refPath = refPath
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

func (s *ReplaceState) WithURLParam() *ReplaceState {
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
