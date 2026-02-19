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
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// redisHistoryTable is a Redis-backed implementation of HistoryTable.
type redisHistoryTable struct {
	client      *redis.Client
	serviceName string
	namespace   string // format: {service}:history
	storage     Table
	ttl         time.Duration
}

// newRedisHistoryTable creates a new Redis-backed history table.
func newRedisHistoryTable(client *redis.Client, serviceName string, storage Table, ttl time.Duration) *redisHistoryTable {
	return &redisHistoryTable{
		client:      client,
		serviceName: serviceName,
		namespace:   serviceName + ":history",
		storage:     storage,
		ttl:         ttl,
	}
}

// historyRecord is the serializable form of RequestedResource for Redis storage.
type historyRecord struct {
	Resource string    `json:"resource"`
	Body     []byte    `json:"body"`
	Response *Response `json:"response,omitempty"`
}

// Get retrieves a request record by the HTTP request.
func (h *redisHistoryTable) Get(req *http.Request) (*RequestedResource, bool) {
	ctx := context.Background()
	key := h.fullKey(h.getKey(req))

	data, err := h.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false
	}
	if err != nil {
		return nil, false
	}

	var record historyRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, false
	}

	return &RequestedResource{
		Resource: record.Resource,
		Body:     record.Body,
		Response: record.Response,
		Request:  req,
		Storage:  h.storage,
	}, true
}

// Set stores a request record.
func (h *redisHistoryTable) Set(resource string, req *http.Request, response *Response) *RequestedResource {
	ctx := context.Background()
	key := h.fullKey(h.getKey(req))

	// Try to get existing record for body reuse
	var existingBody []byte
	if existing, err := h.client.Get(ctx, key).Bytes(); err == nil {
		var existingRecord historyRecord
		if json.Unmarshal(existing, &existingRecord) == nil {
			existingBody = existingRecord.Body
		}
	}

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
	} else if len(existingBody) > 0 {
		body = existingBody
	}

	record := historyRecord{
		Resource: resource,
		Body:     body,
		Response: response,
	}

	data, err := json.Marshal(record)
	if err != nil {
		slog.Error("Error marshaling history record", "error", err)
		return nil
	}

	h.client.Set(ctx, key, data, h.ttl)

	return &RequestedResource{
		Resource: resource,
		Body:     body,
		Request:  req,
		Response: response,
		Storage:  h.storage,
	}
}

// SetResponse updates the response for an existing request record.
func (h *redisHistoryTable) SetResponse(req *http.Request, response *Response) {
	ctx := context.Background()
	key := h.fullKey(h.getKey(req))

	data, err := h.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		slog.Info(fmt.Sprintf("Request for URL %s not found. Cannot set response", req.URL.String()))
		return
	}
	if err != nil {
		slog.Error("Error getting history record", "error", err)
		return
	}

	var record historyRecord
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

	h.client.Set(ctx, key, newData, h.ttl)
}

// Data returns all request records.
// Note: This scans all keys with the namespace prefix, which can be slow for large datasets.
func (h *redisHistoryTable) Data() map[string]*RequestedResource {
	ctx := context.Background()
	pattern := h.namespace + ":*"

	result := make(map[string]*RequestedResource)
	iter := h.client.Scan(ctx, 0, pattern, 0).Iterator()

	for iter.Next(ctx) {
		fullKey := iter.Val()
		// Extract the key part after the namespace
		key := strings.TrimPrefix(fullKey, h.namespace+":")

		data, err := h.client.Get(ctx, fullKey).Bytes()
		if err != nil {
			continue
		}

		var record historyRecord
		if err := json.Unmarshal(data, &record); err != nil {
			continue
		}

		result[key] = &RequestedResource{
			Resource: record.Resource,
			Body:     record.Body,
			Response: record.Response,
			Storage:  h.storage,
		}
	}

	return result
}

// Clear removes all history records.
func (h *redisHistoryTable) Clear() {
	ctx := context.Background()
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

func (h *redisHistoryTable) fullKey(key string) string {
	return h.namespace + ":" + key
}

func (h *redisHistoryTable) getKey(req *http.Request) string {
	return strings.Join([]string{req.Method, req.URL.String()}, ":")
}
