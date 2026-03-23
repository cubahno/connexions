package db

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// redisHistoryTable is a Redis-backed implementation of HistoryTable.
type redisHistoryTable struct {
	client    *redis.Client
	namespace string // format: {service}:history
	ttl       time.Duration
}

// redisHistoryRecord is the serializable form of HistoryEntry for Redis storage.
type redisHistoryRecord struct {
	ID         string           `json:"id"`
	Resource   string           `json:"resource"`
	Method     string           `json:"method"`
	URL        string           `json:"url"`
	Body       []byte           `json:"body"`
	Response   *HistoryResponse `json:"response,omitempty"`
	RemoteAddr string           `json:"remoteAddr,omitempty"`
	CreatedAt  time.Time        `json:"createdAt"`
}

// newRedisHistoryTable creates a new Redis-backed history table.
func newRedisHistoryTable(client *redis.Client, namespace string, ttl time.Duration) *redisHistoryTable {
	return &redisHistoryTable{
		client:    client,
		namespace: namespace,
		ttl:       ttl,
	}
}

// Get retrieves the latest request record matching the HTTP request's method and URL.
func (h *redisHistoryTable) Get(ctx context.Context, req *http.Request) (*HistoryEntry, bool) {
	id, err := h.client.Get(ctx, h.latestKey(req)).Result()
	if errors.Is(err, redis.Nil) || err != nil {
		return nil, false
	}

	data, err := h.client.Get(ctx, h.entryKey(id)).Bytes()
	if errors.Is(err, redis.Nil) || err != nil {
		return nil, false
	}

	var record redisHistoryRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, false
	}

	return &HistoryEntry{
		ID:         record.ID,
		Resource:   record.Resource,
		Body:       record.Body,
		Response:   record.Response,
		Request:    req,
		RemoteAddr: record.RemoteAddr,
		CreatedAt:  record.CreatedAt,
	}, true
}

// Set stores a request record with a unique ID.
func (h *redisHistoryTable) Set(ctx context.Context, resource string, req *http.Request, response *HistoryResponse) *HistoryEntry {
	id := fmt.Sprintf("%d", h.client.Incr(ctx, h.namespace+":counter").Val())

	// Extract the body from the request
	var body []byte
	if req.Body != nil && req.Body != http.NoBody {
		var err error
		body, err = io.ReadAll(req.Body)
		if err != nil {
			slog.Error("Error reading request body", "error", err)
			body = []byte{}
		}
		// Restore the body so it can be read by subsequent handlers
		req.Body = io.NopCloser(bytes.NewBuffer(body))
	}

	now := time.Now().UTC()
	record := redisHistoryRecord{
		ID:         id,
		Resource:   resource,
		Method:     req.Method,
		URL:        req.URL.String(),
		Body:       body,
		Response:   response,
		RemoteAddr: req.RemoteAddr,
		CreatedAt:  now,
	}

	data, err := json.Marshal(record)
	if err != nil {
		slog.Error("Error marshaling history record", "error", err)
		return nil
	}

	// Pipeline both SETs into a single round-trip.
	pipe := h.client.Pipeline()
	pipe.Set(ctx, h.entryKey(id), data, h.ttl)
	pipe.Set(ctx, h.latestKey(req), id, h.ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		slog.Error("Error saving history record", "error", err)
	}

	return &HistoryEntry{
		ID:         id,
		Resource:   resource,
		Body:       body,
		Request:    req,
		Response:   response,
		RemoteAddr: req.RemoteAddr,
		CreatedAt:  now,
	}
}

// SetResponse updates the response for the latest request record matching the HTTP request.
func (h *redisHistoryTable) SetResponse(ctx context.Context, req *http.Request, response *HistoryResponse) {
	id, err := h.client.Get(ctx, h.latestKey(req)).Result()
	if errors.Is(err, redis.Nil) {
		slog.Info(fmt.Sprintf("Request for URL %s not found. Cannot set response", req.URL.String()))
		return
	}
	if err != nil {
		slog.Error("Error getting latest ID", "error", err)
		return
	}

	entryKey := h.entryKey(id)
	data, err := h.client.Get(ctx, entryKey).Bytes()
	if err != nil {
		slog.Error("Error getting history record", "error", err)
		return
	}

	var record redisHistoryRecord
	if err := json.Unmarshal(data, &record); err != nil {
		slog.Error("Error unmarshaling history record", "error", err)
		return
	}

	record.Response = response

	newData, err := json.Marshal(record)
	if err != nil {
		slog.Error("Error marshaling history record", "error", err)
		return
	}

	h.client.Set(ctx, entryKey, newData, h.ttl)
}

// GetByID retrieves a single history entry by its ID.
func (h *redisHistoryTable) GetByID(ctx context.Context, id string) (*HistoryEntry, bool) {
	data, err := h.client.Get(ctx, h.entryKey(id)).Bytes()
	if err != nil {
		return nil, false
	}

	var record redisHistoryRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, false
	}

	entry := &HistoryEntry{
		ID:         record.ID,
		Resource:   record.Resource,
		Body:       record.Body,
		Response:   record.Response,
		RemoteAddr: record.RemoteAddr,
		CreatedAt:  record.CreatedAt,
	}

	if record.Method != "" || record.URL != "" {
		parsedURL, _ := url.Parse(record.URL)
		entry.Request = &http.Request{
			Method: record.Method,
			URL:    parsedURL,
		}
	}

	return entry, true
}

// Data returns all request records as an ordered log.
func (h *redisHistoryTable) Data(ctx context.Context) []*HistoryEntry {
	pattern := h.namespace + ":entry:*"

	// Collect all keys first, then batch-fetch with MGET.
	var keys []string
	iter := h.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if len(keys) == 0 {
		return nil
	}

	vals, err := h.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil
	}

	entries := make([]*HistoryEntry, 0, len(vals))
	for _, val := range vals {
		str, ok := val.(string)
		if !ok || str == "" {
			continue
		}

		var record redisHistoryRecord
		if err := json.Unmarshal([]byte(str), &record); err != nil {
			continue
		}

		entry := &HistoryEntry{
			ID:         record.ID,
			Resource:   record.Resource,
			Body:       record.Body,
			Response:   record.Response,
			RemoteAddr: record.RemoteAddr,
			CreatedAt:  record.CreatedAt,
		}

		if record.Method != "" || record.URL != "" {
			parsedURL, _ := url.Parse(record.URL)
			entry.Request = &http.Request{
				Method: record.Method,
				URL:    parsedURL,
			}
		}

		entries = append(entries, entry)
	}

	// Sort by CreatedAt for stable ordering
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].CreatedAt.Before(entries[j].CreatedAt)
	})

	return entries
}

// Len returns the number of history entries.
func (h *redisHistoryTable) Len(ctx context.Context) int {
	pattern := h.namespace + ":entry:*"
	var count int
	iter := h.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		count++
	}
	return count
}

// Clear removes all history records.
func (h *redisHistoryTable) Clear(ctx context.Context) {
	pattern := h.namespace + ":*"

	iter := h.client.Scan(ctx, 0, pattern, 0).Iterator()
	var keys []string

	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if len(keys) > 0 {
		h.client.Del(ctx, keys...)
	}
}

func (h *redisHistoryTable) entryKey(id string) string {
	return h.namespace + ":entry:" + id
}

func (h *redisHistoryTable) latestKey(req *http.Request) string {
	return h.namespace + ":latest:" + strings.Join([]string{req.Method, req.URL.String()}, ":")
}
