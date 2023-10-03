//go:build !integration

package api

import (
	"bytes"
	"github.com/cubahno/connexions/config"
	assert2 "github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestNewRouter(t *testing.T) {
	assert := assert2.New(t)
	config := &config.Config{}
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

func TestRouter(t *testing.T) {
	assert := assert2.New(t)

	t.Run("AddService", func(t *testing.T) {
		router := new(Router)
		router.AddService(&ServiceItem{Name: "a"})
		router.AddService(&ServiceItem{Name: "b"})

		assert.Len(router.services, 2)
		assert.Equal(&ServiceItem{Name: "a"}, router.services["a"])
		assert.Equal(&ServiceItem{Name: "b"}, router.services["b"])
	})

	t.Run("SetServices", func(t *testing.T) {
		router := new(Router)
		router.SetServices(map[string]*ServiceItem{
			"a": {Name: "a"},
			"b": {Name: "b"},
		})

		assert.Len(router.services, 2)
		assert.Equal(&ServiceItem{Name: "a"}, router.services["a"])
		assert.Equal(&ServiceItem{Name: "b"}, router.services["b"])
	})

	t.Run("GetServices", func(t *testing.T) {
		router := new(Router)
		router.services = map[string]*ServiceItem{
			"a": {Name: "a"},
			"b": {Name: "b"},
		}
		assert.Equal(map[string]*ServiceItem{
			"a": {Name: "a"},
			"b": {Name: "b"},
		}, router.GetServices())
	})

	t.Run("RemoveService", func(t *testing.T) {
		router := new(Router)
		router.services = map[string]*ServiceItem{
			"a": {Name: "a"},
			"b": {Name: "b"},
		}
		router.RemoveService("a")
		assert.Len(router.services, 1)
		assert.Equal(&ServiceItem{Name: "b"}, router.services["b"])
	})

	t.Run("SetContexts", func(t *testing.T) {
		router := new(Router)
		router.SetContexts(
			map[string]map[string]any{
				"a": {"a1": "v1"},
				"b": {"b1": "v2"},
				"c": {"c1": "v1", "c2": "v2"},
			},
			[]map[string]string{
				{"b": "b1"},
				{"a": "a1"},
				{"c": ""},
			},
		)

		assert.Equal(map[string]map[string]any{
			"a": {"a1": "v1"},
			"b": {"b1": "v2"},
			"c": {"c1": "v1", "c2": "v2"},
		}, router.contexts)

		assert.Equal([]map[string]string{
			{"a": "a1"},
			{"b": "b1"},
			{"c": ""},
		}, router.defaultContexts)
	})

	t.Run("GetContexts", func(t *testing.T) {
		router := new(Router)
		router.contexts = map[string]map[string]any{
			"a": {"k1": "v1"},
			"b": {"k2": "v2"},
		}
		assert.Equal(map[string]map[string]any{
			"a": {"k1": "v1"},
			"b": {"k2": "v2"},
		}, router.GetContexts())
	})

	t.Run("GetDefaultContexts", func(t *testing.T) {
		router := new(Router)
		router.defaultContexts = []map[string]string{{"k3": "v3"}}
		assert.Equal([]map[string]string{{"k3": "v3"}}, router.GetDefaultContexts())
	})

	t.Run("RemoveContext", func(t *testing.T) {
		router := new(Router)
		router.contexts = map[string]map[string]any{
			"a": {"k1": "v1"},
			"b": {"k2": "v2"},
		}
		router.RemoveContext("a")
		assert.Len(router.contexts, 1)
		assert.Equal(map[string]map[string]any{"b": {"k2": "v2"}}, router.contexts)
	})

	assert.True(true)
}
