//go:build unit

package connexions

import (
	"bytes"
	"errors"
	assert2 "github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestNewRouter(t *testing.T) {
	assert := assert2.New(t)
	config := &Config{}
	router := NewRouter(config)

	assert.NotNil(router)
	assert.NotNil(router.Mux)
	assert.Equal(config, router.Config)
}

func TestGetJSONPayload(t *testing.T) {
	assert := assert2.New(t)

	type person struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	type test struct {
		payload  string
		expected *person
		err      bool
	}

	tests := []test{
		{
			payload:  `{"name": "Jane Doe", "age": 30}`,
			expected: &person{"Jane Doe", 30},
			err:      false,
		},
		{
			payload:  `invalid JSON`,
			expected: nil,
			err:      true,
		},
	}

	for _, tt := range tests {
		req, err := http.NewRequest("POST", "/", bytes.NewBufferString(tt.payload))
		if err != nil {
			t.Errorf("NewRequest failed: %v", err)
			continue
		}

		payload, err := GetJSONPayload[person](req)
		if tt.err {
			assert.NotNil(err)
		} else {
			assert.Equal(tt.expected, payload)
		}
	}
}

func TestNewErrorMessage(t *testing.T) {
	assert := assert2.New(t)
	err := errors.New("some-error")
	res := NewErrorMessage(err)
	expected := &ErrorMessage{
		Message: "some-error",
	}
	assert.Equal(expected, res)
}

func TestRouter(t *testing.T) {
	assert := assert2.New(t)

	t.Run("RemoveContext", func(t *testing.T) {
		router := new(Router)
		router.Contexts = map[string]map[string]any{
			"a": {"k1": "v1"},
			"b": {"k2": "v2"},
		}
		router.RemoveContext("a")
		assert.Len(router.Contexts, 1)
		assert.Equal(map[string]map[string]any{"b": {"k2": "v2"}}, router.Contexts)
	})

	assert.True(true)
}
