package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/cubahno/connexions_plugin"
	"github.com/go-chi/chi/v5/middleware"
	assert2 "github.com/stretchr/testify/assert"
)

func TestNewCurrentRequestStorage(t *testing.T) {
	assert := assert2.New(t)
	res := NewCurrentRequestStorage(1 * time.Millisecond)

	assert.NotNil(res)
	assert.Equal(0, len(res.getData()))
}

func TestStartResetTicker(t *testing.T) {
	assert := assert2.New(t)
	storage := &CurrentRequestStorage{
		data: map[string]*connexions_plugin.RequestedResource{
			"foo": {},
			"bar": {},
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	startResetTicker(ctx, storage, 100*time.Millisecond)

	time.Sleep(110 * time.Millisecond)
	assert.Equal(0, len(storage.getData()))
}

func TestCurrentRequestStorage(t *testing.T) {
	assert := assert2.New(t)

	t.Run("Get", func(t *testing.T) {
		storage := &CurrentRequestStorage{
			data: map[string]*connexions_plugin.RequestedResource{
				"foo": {Resource: "Foo"},
			},
		}
		assert.Equal("Foo", storage.getData()["foo"].Resource)
	})

	t.Run("Set", func(t *testing.T) {
		payload := map[string]any{
			"foo": "bar",
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("PATCH", "/foo/1", bytes.NewBuffer(body))
		req = req.WithContext(context.WithValue(req.Context(), middleware.RequestIDKey, "foo-123"))
		req.Header.Set("authorization", "Bearer 123")
		res := &connexions_plugin.HistoryResponse{
			StatusCode:     204,
			Data:           body,
			IsFromUpstream: true,
		}

		storage := NewCurrentRequestStorage(100 * time.Millisecond)
		storage.Set("/foo/{id}", req, res)

		assert.Equal(1, len(storage.getData()))

		item := storage.getData()["foo-123"]
		assert.Equal("/foo/{id}", item.Resource)
		assert.Equal("PATCH", item.Method)
		assert.Equal(&url.URL{
			Path: "/foo/1",
		}, item.URL)
		assert.Equal(map[string][]string{
			"Authorization": {"Bearer 123"},
		}, item.Headers)
		assert.Equal(body, item.Body)

		assert.Equal(204, item.Response.StatusCode)
		assert.Equal(body, item.Response.Data)
		assert.True(item.Response.IsFromUpstream)
	})

	t.Run("SetResponse", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/foo/1", nil)
		req = req.WithContext(context.WithValue(req.Context(), middleware.RequestIDKey, "foo-123"))
		res := &connexions_plugin.HistoryResponse{
			StatusCode: 200,
			Data:       []byte(`{"message": "Hello, World!"}`),
		}

		storage := NewCurrentRequestStorage(100 * time.Millisecond)
		storage.Set("/foo/{id}", req, nil)
		storage.SetResponse(req, res)

		item := storage.getData()["foo-123"]
		assert.Equal(200, item.Response.StatusCode)
		assert.Equal([]byte(`{"message": "Hello, World!"}`), item.Response.Data)
	})

	t.Run("Clear", func(t *testing.T) {
		req1, _ := http.NewRequest("GET", "/foo/1", nil)
		req1 = req1.WithContext(context.WithValue(req1.Context(), middleware.RequestIDKey, "foo-123"))

		req2, _ := http.NewRequest("GET", "/foo/2", nil)
		req2 = req1.WithContext(context.WithValue(req2.Context(), middleware.RequestIDKey, "foo-234"))

		storage := NewCurrentRequestStorage(100 * time.Millisecond)
		storage.Set("/foo/{id}", req1, nil)
		storage.Set("/bar/{id}", req2, nil)

		assert.Equal(2, len(storage.getData()))

		storage.Clear()
		assert.Equal(0, len(storage.getData()))
	})
}
