package db

import (
	"bytes"
	"context"
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

// newTestHistoryTable creates a history table with a backing memory table for testing.
func newTestHistoryTable(clearTimeout time.Duration) *memoryHistoryTable {
	table := newMemoryTable()
	return newMemoryHistoryTable(table, clearTimeout)
}

func TestMemoryHistoryTable_Get(t *testing.T) {
	assert := assert2.New(t)
	ctx := context.Background()

	t.Run("get existing request", func(t *testing.T) {
		h := newTestHistoryTable(0)
		// Seed data via Set instead of direct access
		req, _ := http.NewRequest("GET", "/foo/1", nil)
		h.Set(ctx, "Foo", req, nil)

		resource, ok := h.Get(ctx, req)

		assert.True(ok)
		assert.Equal("Foo", resource.Resource)
	})

	t.Run("get non-existing request", func(t *testing.T) {
		h := newTestHistoryTable(0)

		req, _ := http.NewRequest("GET", "/foo/1", nil)
		resource, ok := h.Get(ctx, req)

		assert.False(ok)
		assert.Nil(resource)
	})
}

func TestMemoryHistoryTable_Set(t *testing.T) {
	assert := assert2.New(t)
	ctx := context.Background()

	t.Run("set new request", func(t *testing.T) {
		h := newTestHistoryTable(0)

		req, _ := http.NewRequest("POST", "/foo/1", bytes.NewBufferString(`{"name":"test"}`))
		result := h.Set(ctx, "/foo/{id}", req, nil)

		assert.Equal("/foo/{id}", result.Resource)
		assert.Equal(`{"name":"test"}`, string(result.Body))
	})

	t.Run("set with response", func(t *testing.T) {
		h := newTestHistoryTable(0)

		req, _ := http.NewRequest("GET", "/foo/1", nil)
		response := &HistoryResponse{Data: []byte("response"), StatusCode: 200}
		result := h.Set(ctx, "/foo/{id}", req, response)

		assert.Equal(response, result.Response)
	})

	t.Run("reuse body from existing record", func(t *testing.T) {
		h := newTestHistoryTable(0)

		// First request with body
		req1, _ := http.NewRequest("POST", "/foo/1", bytes.NewBufferString(`{"name":"test"}`))
		h.Set(ctx, "/foo/{id}", req1, nil)

		// Second request without body (same key)
		req2, _ := http.NewRequest("POST", "/foo/1", nil)
		result := h.Set(ctx, "/foo/{id}", req2, nil)

		assert.Equal(`{"name":"test"}`, string(result.Body))
	})

	t.Run("set with error reading body", func(t *testing.T) {
		h := newTestHistoryTable(0)

		req, _ := http.NewRequest("POST", "/foo/1", io.NopCloser(&errorReader{}))
		result := h.Set(ctx, "/foo/{id}", req, nil)

		assert.Equal([]byte{}, result.Body)
	})
}

func TestMemoryHistoryTable_SetResponse(t *testing.T) {
	assert := assert2.New(t)
	ctx := context.Background()

	t.Run("set response for existing request", func(t *testing.T) {
		h := newTestHistoryTable(0)

		req, _ := http.NewRequest("GET", "/foo/1", nil)
		h.Set(ctx, "/foo/{id}", req, nil)

		response := &HistoryResponse{Data: []byte("response"), StatusCode: 200}
		h.SetResponse(ctx, req, response)

		record, _ := h.Get(ctx, req)
		assert.Equal(response, record.Response)
	})

	t.Run("set response for non-existing request", func(t *testing.T) {
		h := newTestHistoryTable(0)

		req, _ := http.NewRequest("GET", "/foo/1", nil)
		response := &HistoryResponse{Data: []byte("response"), StatusCode: 200}

		// Should not panic, just log
		h.SetResponse(ctx, req, response)
	})

	t.Run("set response for invalid record type", func(t *testing.T) {
		h := newTestHistoryTable(0)

		req, _ := http.NewRequest("GET", "/foo/1", nil)
		// Inject a non-HistoryEntry value directly into the table
		h.table.Set(ctx, "GET:/foo/1", "not a history entry", 0)

		response := &HistoryResponse{Data: []byte("response"), StatusCode: 200}
		// Should not panic, just return early
		h.SetResponse(ctx, req, response)
	})
}

func TestMemoryHistoryTable_Data(t *testing.T) {
	assert := assert2.New(t)
	ctx := context.Background()

	h := newTestHistoryTable(0)

	req, _ := http.NewRequest("GET", "/foo/1", nil)
	h.Set(ctx, "/foo/{id}", req, nil)

	data := h.Data(ctx)

	assert.Len(data, 1)
	assert.Contains(data, "GET:/foo/1")
}

func TestMemoryHistoryTable_Clear(t *testing.T) {
	assert := assert2.New(t)
	ctx := context.Background()

	h := newTestHistoryTable(0)

	req, _ := http.NewRequest("GET", "/foo/1", nil)
	h.Set(ctx, "/foo/{id}", req, nil)

	h.Clear(ctx)

	assert.Empty(h.Data(ctx))
}

func TestMemoryHistoryTable_AutoClear(t *testing.T) {
	assert := assert2.New(t)
	ctx := context.Background()

	h := newTestHistoryTable(50 * time.Millisecond)
	defer h.cancel()

	req := &http.Request{Method: "GET", URL: &url.URL{Path: "/foo/1"}}
	h.Set(ctx, "/foo/{id}", req, nil)

	assert.Len(h.Data(ctx), 1)

	// Wait for auto-clear
	time.Sleep(100 * time.Millisecond)

	assert.Empty(h.Data(ctx))
}

func TestMemoryHistoryTable_Cancel(t *testing.T) {
	assert := assert2.New(t)
	ctx := context.Background()

	h := newTestHistoryTable(50 * time.Millisecond)

	req := &http.Request{Method: "GET", URL: &url.URL{Path: "/foo/1"}}
	h.Set(ctx, "/foo/{id}", req, nil)

	// Cancel should stop the reset ticker
	h.cancel()

	// Wait past the clear timeout
	time.Sleep(100 * time.Millisecond)

	// Data should still be there since ticker was cancelled
	assert.Len(h.Data(ctx), 1)
}
