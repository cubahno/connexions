package db

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	assert2 "github.com/stretchr/testify/assert"
)

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
		h.Set(ctx, "Foo", &HistoryRequest{Method: "GET", URL: "/foo/1"}, nil)

		req, _ := http.NewRequest("GET", "/foo/1", nil)
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

	t.Run("get returns latest entry", func(t *testing.T) {
		h := newTestHistoryTable(0)
		histReq := &HistoryRequest{Method: "GET", URL: "/foo/1"}

		h.Set(ctx, "First", histReq, nil)
		h.Set(ctx, "Second", histReq, nil)

		req, _ := http.NewRequest("GET", "/foo/1", nil)
		resource, ok := h.Get(ctx, req)
		assert.True(ok)
		assert.Equal("Second", resource.Resource)
	})
}

func TestMemoryHistoryTable_Set(t *testing.T) {
	assert := assert2.New(t)
	ctx := context.Background()

	t.Run("set new request", func(t *testing.T) {
		h := newTestHistoryTable(0)

		result := h.Set(ctx, "/foo/{id}", &HistoryRequest{
			Method: "POST",
			URL:    "/foo/1",
			Body:   []byte(`{"name":"test"}`),
		}, nil)

		assert.Equal("/foo/{id}", result.Resource)
		assert.Equal(`{"name":"test"}`, string(result.Request.Body))
		assert.NotEmpty(result.ID)
	})

	t.Run("set with response", func(t *testing.T) {
		h := newTestHistoryTable(0)

		response := &HistoryResponse{Body: []byte("response"), StatusCode: 200}
		result := h.Set(ctx, "/foo/{id}", &HistoryRequest{Method: "GET", URL: "/foo/1"}, response)

		assert.Equal(response, result.Response)
	})

	t.Run("multiple sets to same endpoint create unique entries", func(t *testing.T) {
		h := newTestHistoryTable(0)

		e1 := h.Set(ctx, "/foo/{id}", &HistoryRequest{Method: "POST", URL: "/foo/1", Body: []byte(`{"a":"1"}`)}, nil)
		e2 := h.Set(ctx, "/foo/{id}", &HistoryRequest{Method: "POST", URL: "/foo/1", Body: []byte(`{"a":"2"}`)}, nil)

		assert.NotEqual(e1.ID, e2.ID)
		assert.Len(h.Data(ctx), 2)
	})
}

func TestMemoryHistoryTable_GetByID(t *testing.T) {
	assert := assert2.New(t)
	ctx := context.Background()

	t.Run("returns entry by ID", func(t *testing.T) {
		h := newTestHistoryTable(0)
		entry := h.Set(ctx, "/foo", &HistoryRequest{Method: "GET", URL: "/foo"}, &HistoryResponse{StatusCode: 200})

		got, ok := h.GetByID(ctx, entry.ID)
		assert.True(ok)
		assert.Equal(entry.ID, got.ID)
	})

	t.Run("returns false for unknown ID", func(t *testing.T) {
		h := newTestHistoryTable(0)
		_, ok := h.GetByID(ctx, "nonexistent")
		assert.False(ok)
	})
}

func TestMemoryHistoryTable_SetResponse(t *testing.T) {
	assert := assert2.New(t)
	ctx := context.Background()

	t.Run("set response for existing request", func(t *testing.T) {
		h := newTestHistoryTable(0)

		histReq := &HistoryRequest{Method: "GET", URL: "/foo/1"}
		h.Set(ctx, "/foo/{id}", histReq, nil)

		response := &HistoryResponse{Body: []byte("response"), StatusCode: 200}
		h.SetResponse(ctx, histReq, response)

		req, _ := http.NewRequest("GET", "/foo/1", nil)
		record, _ := h.Get(ctx, req)
		assert.Equal(response, record.Response)
	})

	t.Run("set response for non-existing request", func(t *testing.T) {
		h := newTestHistoryTable(0)

		response := &HistoryResponse{Body: []byte("response"), StatusCode: 200}

		// Should not panic, just log
		h.SetResponse(ctx, &HistoryRequest{Method: "GET", URL: "/foo/1"}, response)
	})

	t.Run("set response updates latest entry only", func(t *testing.T) {
		h := newTestHistoryTable(0)

		histReq := &HistoryRequest{Method: "GET", URL: "/foo/1"}
		h.Set(ctx, "/foo/{id}", histReq, &HistoryResponse{StatusCode: 100})
		h.Set(ctx, "/foo/{id}", histReq, nil)

		h.SetResponse(ctx, histReq, &HistoryResponse{StatusCode: 200})

		entries := h.Data(ctx)
		assert.Len(entries, 2)
		// First entry should keep its original response
		assert.Equal(100, entries[0].Response.StatusCode)
		// Second (latest) entry should have the updated response
		assert.Equal(200, entries[1].Response.StatusCode)
	})
}

func TestMemoryHistoryTable_Data(t *testing.T) {
	assert := assert2.New(t)
	ctx := context.Background()

	h := newTestHistoryTable(0)

	h.Set(ctx, "/foo/{id}", &HistoryRequest{Method: "GET", URL: "/foo/1"}, nil)

	data := h.Data(ctx)

	assert.Len(data, 1)
	assert.Equal("/foo/{id}", data[0].Resource)
	assert.NotEmpty(data[0].ID)
}

func TestMemoryHistoryTable_Len(t *testing.T) {
	assert := assert2.New(t)
	ctx := context.Background()

	h := newTestHistoryTable(0)
	assert.Equal(0, h.Len(ctx))

	histReq := &HistoryRequest{Method: "GET", URL: "/foo/1"}
	h.Set(ctx, "/foo/{id}", histReq, nil)
	assert.Equal(1, h.Len(ctx))

	h.Set(ctx, "/foo/{id}", histReq, nil)
	assert.Equal(2, h.Len(ctx))
}

func TestMemoryHistoryTable_Clear(t *testing.T) {
	assert := assert2.New(t)
	ctx := context.Background()

	h := newTestHistoryTable(0)

	h.Set(ctx, "/foo/{id}", &HistoryRequest{Method: "GET", URL: "/foo/1"}, nil)

	h.Clear(ctx)

	assert.Empty(h.Data(ctx))
	assert.Equal(0, h.Len(ctx))
}

func TestMemoryHistoryTable_AutoClear(t *testing.T) {
	assert := assert2.New(t)
	ctx := context.Background()

	h := newTestHistoryTable(50 * time.Millisecond)
	defer h.cancel()

	h.Set(ctx, "/foo/{id}", &HistoryRequest{Method: "GET", URL: "/foo/1"}, nil)

	assert.Equal(1, h.Len(ctx))

	// Wait for auto-clear
	time.Sleep(100 * time.Millisecond)

	assert.Empty(h.Data(ctx))
}

func TestMemoryHistoryTable_Cancel(t *testing.T) {
	assert := assert2.New(t)
	ctx := context.Background()

	h := newTestHistoryTable(50 * time.Millisecond)

	h.Set(ctx, "/foo/{id}", &HistoryRequest{Method: "GET", URL: "/foo/1"}, nil)

	// Cancel should stop the reset ticker
	h.cancel()

	// Wait past the clear timeout
	time.Sleep(100 * time.Millisecond)

	// Data should still be there since ticker was cancelled
	assert.Equal(1, h.Len(ctx))
}

func TestFlattenHeaders(t *testing.T) {
	assert := assert2.New(t)

	t.Run("nil header", func(t *testing.T) {
		assert.Nil(FlattenHeaders(nil))
	})

	t.Run("empty header", func(t *testing.T) {
		assert.Nil(FlattenHeaders(http.Header{}))
	})

	t.Run("single values sorted", func(t *testing.T) {
		h := http.Header{
			"Content-Type": {"application/json"},
			"Accept":       {"text/html"},
		}
		result := FlattenHeaders(h)
		assert.Equal([]string{
			"Accept: text/html",
			"Content-Type: application/json",
		}, result)
	})

	t.Run("multi values joined", func(t *testing.T) {
		h := http.Header{
			"Accept": {"text/html", "application/json"},
		}
		result := FlattenHeaders(h)
		assert.Equal([]string{"Accept: text/html, application/json"}, result)
	})
}

// lookupKey is tested indirectly through Get (uses *http.Request) and Set (uses *HistoryRequest)
// to verify they produce the same key.
func TestMemoryHistoryTable_LookupKeyConsistency(t *testing.T) {
	assert := assert2.New(t)
	ctx := context.Background()

	h := newTestHistoryTable(0)

	// Set via HistoryRequest
	h.Set(ctx, "/test", &HistoryRequest{Method: "GET", URL: "/test?q=1"}, &HistoryResponse{StatusCode: 200})

	// Get via *http.Request with same method+URL
	req := &http.Request{Method: "GET", URL: &url.URL{Path: "/test", RawQuery: "q=1"}}
	entry, ok := h.Get(ctx, req)
	assert.True(ok)
	assert.Equal(200, entry.Response.StatusCode)
}
