package db

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// redisTable is a Redis-backed implementation of Table.
// Keys are namespaced as {service}:{tableName}:{key}
type redisTable struct {
	client    *redis.Client
	namespace string // format: {service}:{tableName}
}

// newRedisTable creates a new Redis-backed table.
func newRedisTable(client *redis.Client, serviceName, tableName string) *redisTable {
	return &redisTable{
		client:    client,
		namespace: serviceName + ":" + tableName,
	}
}

// Get retrieves a value by key.
func (t *redisTable) Get(ctx context.Context, key string) (any, bool) {
	fullKey := t.fullKey(key)

	data, err := t.client.Get(ctx, fullKey).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false
	}
	if err != nil {
		return nil, false
	}

	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, false
	}
	return value, true
}

// Set stores a value with the given key.
// If ttl is 0, the value never expires.
func (t *redisTable) Set(ctx context.Context, key string, value any, ttl time.Duration) {
	fullKey := t.fullKey(key)

	data, err := json.Marshal(value)
	if err != nil {
		return
	}

	t.client.Set(ctx, fullKey, data, ttl)
}

// Delete removes a value by key.
func (t *redisTable) Delete(ctx context.Context, key string) {
	fullKey := t.fullKey(key)
	t.client.Del(ctx, fullKey)
}

// Data returns a copy of all data in the table.
// Note: This scans all keys with the namespace prefix, which can be slow for large datasets.
func (t *redisTable) Data(ctx context.Context) map[string]any {
	pattern := t.namespace + ":*"
	prefix := t.namespace + ":"

	// Collect all keys first, then batch-fetch with MGET.
	var keys []string
	iter := t.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if len(keys) == 0 {
		return make(map[string]any)
	}

	vals, err := t.client.MGet(ctx, keys...).Result()
	if err != nil {
		return make(map[string]any)
	}

	result := make(map[string]any, len(keys))
	for i, val := range vals {
		str, ok := val.(string)
		if !ok || str == "" {
			continue
		}

		var value any
		if err := json.Unmarshal([]byte(str), &value); err != nil {
			continue
		}

		key := strings.TrimPrefix(keys[i], prefix)
		result[key] = value
	}

	return result
}

// Clear removes all data from the table.
func (t *redisTable) Clear(ctx context.Context) {
	pattern := t.namespace + ":*"

	iter := t.client.Scan(ctx, 0, pattern, 0).Iterator()
	var keys []string

	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if len(keys) > 0 {
		t.client.Del(ctx, keys...)
	}
}

func (t *redisTable) fullKey(key string) string {
	return t.namespace + ":" + key
}
