package db

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func newTestRedisHistory(t *testing.T) (*redisHistoryTable, *miniredis.Miniredis) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	history := newRedisHistoryTable(client, "test:history", 5*time.Minute)
	return history, mr
}

func TestRedisHistory_Get(t *testing.T) {
	ctx := context.Background()

	t.Run("get existing entry", func(t *testing.T) {
		history, _ := newTestRedisHistory(t)
		req := &http.Request{Method: "GET", URL: &url.URL{Path: "/users/123"}}
		history.Set(ctx, "/users/123", req, &HistoryResponse{
			StatusCode: 200,
			Data:       []byte(`{"id":123}`),
		})

		entry, ok := history.Get(ctx, req)
		assert.True(t, ok)
		assert.Equal(t, "/users/123", entry.Resource)
		assert.Equal(t, 200, entry.Response.StatusCode)
	})

	t.Run("get non-existing entry", func(t *testing.T) {
		history, _ := newTestRedisHistory(t)
		req := &http.Request{Method: "GET", URL: &url.URL{Path: "/notfound"}}

		entry, ok := history.Get(ctx, req)
		assert.False(t, ok)
		assert.Nil(t, entry)
	})

	t.Run("get with invalid json returns false", func(t *testing.T) {
		history, mr := newTestRedisHistory(t)
		_ = mr.Set("test:history:GET:/bad", "not-valid-json{")

		req := &http.Request{Method: "GET", URL: &url.URL{Path: "/bad"}}
		entry, ok := history.Get(ctx, req)
		assert.False(t, ok)
		assert.Nil(t, entry)
	})
}

func TestRedisHistory_Set(t *testing.T) {
	ctx := context.Background()

	t.Run("set with body", func(t *testing.T) {
		history, _ := newTestRedisHistory(t)
		body := `{"name":"test"}`
		req := &http.Request{
			Method: "POST",
			URL:    &url.URL{Path: "/users"},
			Body:   io.NopCloser(strings.NewReader(body)),
		}

		entry := history.Set(ctx, "/users", req, &HistoryResponse{StatusCode: 201})

		assert.Equal(t, "/users", entry.Resource)
		assert.Equal(t, []byte(body), entry.Body)
		assert.Equal(t, 201, entry.Response.StatusCode)
	})

	t.Run("set without body", func(t *testing.T) {
		history, _ := newTestRedisHistory(t)
		req := &http.Request{Method: "GET", URL: &url.URL{Path: "/health"}}

		entry := history.Set(ctx, "/health", req, nil)

		assert.Equal(t, "/health", entry.Resource)
		assert.Empty(t, entry.Body)
	})

	t.Run("body reuse from existing entry", func(t *testing.T) {
		history, _ := newTestRedisHistory(t)
		body := `{"foo":"bar"}`
		req := &http.Request{
			Method: "POST",
			URL:    &url.URL{Path: "/data"},
			Body:   io.NopCloser(strings.NewReader(body)),
		}
		history.Set(ctx, "/data", req, nil)

		// Second call with no body should reuse existing
		req2 := &http.Request{Method: "POST", URL: &url.URL{Path: "/data"}}
		entry := history.Set(ctx, "/data", req2, &HistoryResponse{StatusCode: 200})

		assert.Equal(t, []byte(body), entry.Body)
	})
}

func TestRedisHistory_SetResponse(t *testing.T) {
	ctx := context.Background()

	t.Run("update existing entry", func(t *testing.T) {
		history, _ := newTestRedisHistory(t)
		req := &http.Request{Method: "GET", URL: &url.URL{Path: "/test"}}
		history.Set(ctx, "/test", req, nil)

		history.SetResponse(ctx, req, &HistoryResponse{
			StatusCode:  200,
			Data:        []byte(`{"ok":true}`),
			ContentType: "application/json",
		})

		entry, ok := history.Get(ctx, req)
		assert.True(t, ok)
		assert.Equal(t, 200, entry.Response.StatusCode)
		assert.Equal(t, "application/json", entry.Response.ContentType)
	})

	t.Run("set response for non-existing entry", func(t *testing.T) {
		history, _ := newTestRedisHistory(t)
		req := &http.Request{Method: "GET", URL: &url.URL{Path: "/missing"}}

		// Should not panic, just log
		history.SetResponse(ctx, req, &HistoryResponse{StatusCode: 200})

		// Entry should still not exist
		_, ok := history.Get(ctx, req)
		assert.False(t, ok)
	})
}

func TestRedisHistory_Data(t *testing.T) {
	ctx := context.Background()

	t.Run("returns all entries", func(t *testing.T) {
		history, _ := newTestRedisHistory(t)
		req1 := &http.Request{Method: "GET", URL: &url.URL{Path: "/a"}}
		req2 := &http.Request{Method: "POST", URL: &url.URL{Path: "/b"}}
		history.Set(ctx, "/a", req1, &HistoryResponse{StatusCode: 200})
		history.Set(ctx, "/b", req2, &HistoryResponse{StatusCode: 201})

		data := history.Data(ctx)

		assert.Len(t, data, 2)
		assert.Equal(t, "/a", data["GET:/a"].Resource)
		assert.Equal(t, "/b", data["POST:/b"].Resource)
	})

	t.Run("empty history", func(t *testing.T) {
		history, _ := newTestRedisHistory(t)

		data := history.Data(ctx)

		assert.Empty(t, data)
	})

	t.Run("skips invalid json entries", func(t *testing.T) {
		history, mr := newTestRedisHistory(t)
		req := &http.Request{Method: "GET", URL: &url.URL{Path: "/valid"}}
		history.Set(ctx, "/valid", req, &HistoryResponse{StatusCode: 200})

		// Inject invalid JSON directly
		_ = mr.Set("test:history:GET:/invalid", "not-valid-json{")

		data := history.Data(ctx)

		// Should only contain the valid entry
		assert.Len(t, data, 1)
		assert.Equal(t, "/valid", data["GET:/valid"].Resource)
	})
}

func TestRedisHistory_Clear(t *testing.T) {
	ctx := context.Background()

	t.Run("clears all entries", func(t *testing.T) {
		history, _ := newTestRedisHistory(t)
		req1 := &http.Request{Method: "GET", URL: &url.URL{Path: "/x"}}
		req2 := &http.Request{Method: "GET", URL: &url.URL{Path: "/y"}}
		history.Set(ctx, "/x", req1, nil)
		history.Set(ctx, "/y", req2, nil)

		history.Clear(ctx)

		data := history.Data(ctx)
		assert.Empty(t, data)
	})

	t.Run("clear empty history", func(t *testing.T) {
		history, _ := newTestRedisHistory(t)
		// Should not panic
		history.Clear(ctx)
	})
}
