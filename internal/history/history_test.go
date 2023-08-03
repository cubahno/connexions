package history

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	assert2 "github.com/stretchr/testify/assert"
)

// errorReader is a reader that always returns an error
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("read error")
}

func TestNewCurrentRequestStorage(t *testing.T) {
	assert := assert2.New(t)
	res := NewCurrentRequestStorage(1 * time.Millisecond)

	assert.NotNil(res)
	assert.Equal(0, len(res.Data()))
}

func TestStartResetTicker(t *testing.T) {
	assert := assert2.New(t)
	storage := &CurrentRequestStorage{
		data: map[string]*RequestedResource{
			"foo": {},
			"bar": {},
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	startResetTicker(ctx, storage, 100*time.Millisecond)

	time.Sleep(110 * time.Millisecond)
	assert.Equal(0, len(storage.Data()))
}

func TestCurrentRequestStorage(t *testing.T) {
	assert := assert2.New(t)

	t.Run("Get", func(t *testing.T) {
		storage := &CurrentRequestStorage{
			data: map[string]*RequestedResource{
				"GET:/foo/1": {Resource: "Foo"},
			},
		}

		req, _ := http.NewRequest("GET", "/foo/1", nil)
		resource, ok := storage.Get(req)
		assert.True(ok)
		assert.Equal("Foo", resource.Resource)
	})

	t.Run("Get not found", func(t *testing.T) {
		storage := &CurrentRequestStorage{
			data: map[string]*RequestedResource{},
		}

		req, _ := http.NewRequest("GET", "/foo/1", nil)
		resource, ok := storage.Get(req)
		assert.False(ok)
		assert.Nil(resource)
	})

	t.Run("Set", func(t *testing.T) {
		payload := map[string]any{
			"foo": "bar",
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("PATCH", "/foo/1", bytes.NewBuffer(body))

		req.Header.Set("authorization", "Bearer 123")
		res := &HistoryResponse{
			StatusCode:     204,
			Data:           body,
			IsFromUpstream: true,
		}

		storage := NewCurrentRequestStorage(100 * time.Millisecond)
		storage.Set("foo", "/foo/{id}", req, res)

		assert.Equal(1, len(storage.Data()))

		item := storage.Data()["PATCH:/foo/1"]
		if item == nil {
			t.Fatal("item not found")
		}
		assert.Equal("/foo/{id}", item.Resource)
		assert.Equal("PATCH", item.Request.Method)
		assert.Equal(&url.URL{
			Path: "/foo/1",
		}, item.Request.URL)
		assert.Equal(
			http.Header{"Authorization": []string{"Bearer 123"}}, item.Request.Header)
		assert.Equal(body, item.Body)

		assert.Equal(204, item.Response.StatusCode)
		assert.Equal(body, item.Response.Data)
		assert.True(item.Response.IsFromUpstream)
	})

	t.Run("SetResponse", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/foo/1", nil)
		res := &HistoryResponse{
			StatusCode: 200,
			Data:       []byte(`{"message": "Hello, World!"}`),
		}

		storage := NewCurrentRequestStorage(100 * time.Millisecond)
		storage.Set("foo", "/foo/{id}", req, nil)
		storage.SetResponse(req, res)

		item := storage.Data()["GET:/foo/1"]
		if item == nil {
			t.Fatal("item not found")
		}
		assert.Equal(200, item.Response.StatusCode)
		assert.Equal([]byte(`{"message": "Hello, World!"}`), item.Response.Data)
	})

	t.Run("Clear", func(t *testing.T) {
		req1, _ := http.NewRequest("GET", "/foo/1", nil)
		req2, _ := http.NewRequest("GET", "/foo/2", nil)

		storage := NewCurrentRequestStorage(100 * time.Millisecond)
		storage.Set("foo", "/foo/{id}", req1, nil)
		storage.Set("foo", "/bar/{id}", req2, nil)

		assert.Equal(2, len(storage.Data()))

		storage.Clear()
		assert.Equal(0, len(storage.Data()))
	})

	t.Run("Cancel", func(t *testing.T) {
		storage := NewCurrentRequestStorage(100 * time.Millisecond)
		storage.Set("foo", "/foo/{id}", &http.Request{
			Method: "GET",
			URL:    &url.URL{Path: "/foo/1"},
		}, nil)

		assert.Equal(1, len(storage.Data()))

		// Cancel should stop the reset ticker
		storage.Cancel()

		// Wait a bit to ensure ticker would have fired if not cancelled
		time.Sleep(150 * time.Millisecond)

		// Data should still be there since ticker was cancelled
		assert.Equal(1, len(storage.Data()))
	})

	t.Run("Cancel with nil cancelFunc", func(t *testing.T) {
		storage := &CurrentRequestStorage{
			data:       make(map[string]*RequestedResource),
			cancelFunc: nil,
		}

		// Should not panic
		storage.Cancel()
	})

	t.Run("Set with existing record and no body", func(t *testing.T) {
		// First, create a request with a body
		payload := map[string]any{"foo": "bar"}
		body, _ := json.Marshal(payload)
		req1, _ := http.NewRequest("POST", "/foo/1", bytes.NewBuffer(body))

		storage := NewCurrentRequestStorage(100 * time.Millisecond)
		storage.Set("foo", "/foo/{id}", req1, nil)

		// Now make a second request to the same endpoint with no body
		req2, _ := http.NewRequest("POST", "/foo/1", nil)
		result := storage.Set("foo", "/foo/{id}", req2, nil)

		// The body from the first request should be reused
		assert.Equal(body, result.Body)
	})

	t.Run("SetResponse for non-existent request", func(t *testing.T) {
		storage := NewCurrentRequestStorage(100 * time.Millisecond)
		req, _ := http.NewRequest("GET", "/nonexistent", nil)
		res := &HistoryResponse{
			StatusCode: 200,
			Data:       []byte(`{"message": "test"}`),
		}

		// This should not panic and should log a message
		storage.SetResponse(req, res)

		// Verify the request was not added
		_, exists := storage.Get(req)
		assert.False(exists)
	})

	t.Run("Set with error reading body", func(t *testing.T) {
		storage := NewCurrentRequestStorage(100 * time.Millisecond)
		req, _ := http.NewRequest("POST", "/foo/1", io.NopCloser(&errorReader{}))

		// This should handle the error gracefully and set an empty body
		result := storage.Set("foo", "/foo/{id}", req, nil)

		// Body should be empty due to read error
		assert.Equal([]byte{}, result.Body)
		assert.NotNil(result.Request)
	})
}

func TestNewMemoryStorage(t *testing.T) {
	assert := assert2.New(t)
	mem := NewMemoryStorage()

	assert.NotNil(mem)
	assert.Equal(0, len(mem.Data()))
}

func TestMemoryStorage_Get(t *testing.T) {
	assert := assert2.New(t)
	mem := NewMemoryStorage()

	mem.Set("foo", "bar")
	res, ok := mem.Get("foo")
	assert.True(ok)
	assert.Equal("bar", res)
}

func TestMemoryStorage_Set(t *testing.T) {
	assert := assert2.New(t)
	mem := NewMemoryStorage()

	mem.Set("foo", "bar")
	assert.Equal(map[string]any{
		"foo": "bar",
	}, mem.Data())
}
