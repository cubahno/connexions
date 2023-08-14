package xs

import (
	"strings"
	"sync"
)

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
