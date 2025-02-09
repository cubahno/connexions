package replacer

import (
	"sync"
)

// ReplaceState is a struct that holds information about the current state of the replace operation.
//
// NamePath is a slice of names of the current element.
// It is used to build a path to the current element.
// For example, "users", "name", "first".
//
// ElementIndex is an index of the current element if required structure to generate is an array.
// IsHeader is a flag that indicates that the current element we're replacing is a header.
// IsPathParam is a flag that indicates that the current element we're replacing is a path parameter.
// ContentType is a content type of the current element.
// IsContentReadOnly is a flag that indicates that the current element we're replacing is a read-only content.
// This value is used only when Scheme has ReadOnly set to true.
//
// IsContentWriteOnly is a flag that indicates that the current element we're replacing is a write-only content.
// This value is used only when Scheme has WriteOnly set to true.
type ReplaceState struct {
	NamePath           []string
	ElementIndex       int
	IsHeader           bool
	IsPathParam        bool
	ContentType        string
	IsContentReadOnly  bool
	IsContentWriteOnly bool
	mu                 sync.Mutex
}

func NewReplaceState(opts ...ReplaceStateOption) *ReplaceState {
	return (&ReplaceState{
		NamePath: []string{},
	}).WithOptions(opts...)
}

func NewReplaceStateWithName(name string) *ReplaceState {
	return NewReplaceState(WithName(name))
}

// NewFrom creates a new ReplaceState instance from the given one.
func (s *ReplaceState) NewFrom(src *ReplaceState) *ReplaceState {
	return &ReplaceState{
		NamePath:           src.NamePath,
		IsHeader:           src.IsHeader,
		IsPathParam:        src.IsPathParam,
		ContentType:        src.ContentType,
		IsContentReadOnly:  src.IsContentReadOnly,
		IsContentWriteOnly: src.IsContentWriteOnly,
	}
}

type ReplaceStateOption func(*ReplaceState)

func (s *ReplaceState) WithOptions(options ...ReplaceStateOption) *ReplaceState {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, opt := range options {
		opt(s)
	}
	return s
}

func WithName(name string) ReplaceStateOption {
	return func(state *ReplaceState) {
		namePath := state.NamePath
		if len(namePath) == 0 {
			namePath = []string{}
		}
		namePath = append(namePath, name)

		state.NamePath = namePath
	}
}

func WithElementIndex(value int) ReplaceStateOption {
	return func(state *ReplaceState) {
		state.ElementIndex = value
	}
}

func WithHeader() ReplaceStateOption {
	return func(state *ReplaceState) {
		state.IsHeader = true
	}
}

func WithPath() ReplaceStateOption {
	return func(state *ReplaceState) {
		state.IsPathParam = true
	}
}

func WithContentType(value string) ReplaceStateOption {
	return func(state *ReplaceState) {
		state.ContentType = value
	}
}

func WithReadOnly() ReplaceStateOption {
	return func(state *ReplaceState) {
		state.IsContentReadOnly = true
	}
}

func WithWriteOnly() ReplaceStateOption {
	return func(state *ReplaceState) {
		state.IsContentWriteOnly = true
	}
}
