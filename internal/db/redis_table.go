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
	ttl       time.Duration
}

// newRedisTable creates a new Redis-backed table.
func newRedisTable(client *redis.Client, serviceName, tableName string, ttl time.Duration) *redisTable {
	return &redisTable{
		client:    client,
		namespace: serviceName + ":" + tableName,
		ttl:       ttl,
	}
}

// Get retrieves a value by key.
func (t *redisTable) Get(key string) (any, bool) {
	ctx := context.Background()
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
func (t *redisTable) Set(key string, value any) {
	ctx := context.Background()
	fullKey := t.fullKey(key)

	data, err := json.Marshal(value)
	if err != nil {
		return
	}

	t.client.Set(ctx, fullKey, data, t.ttl)
}

// Delete removes a value by key.
func (t *redisTable) Delete(key string) {
	ctx := context.Background()
	fullKey := t.fullKey(key)
	t.client.Del(ctx, fullKey)
}

// Data returns a copy of all data in the table.
// Note: This scans all keys with the namespace prefix, which can be slow for large datasets.
func (t *redisTable) Data() map[string]any {
	ctx := context.Background()
	pattern := t.namespace + ":*"

	result := make(map[string]any)
	iter := t.client.Scan(ctx, 0, pattern, 0).Iterator()

	for iter.Next(ctx) {
		fullKey := iter.Val()
		// Extract the key part after the namespace
		key := strings.TrimPrefix(fullKey, t.namespace+":")

		data, err := t.client.Get(ctx, fullKey).Bytes()
		if err != nil {
			continue
		}

		var value any
		if err := json.Unmarshal(data, &value); err != nil {
			continue
		}
		result[key] = value
	}

	return result
}

// Clear removes all data from the table.
func (t *redisTable) Clear() {
	ctx := context.Background()
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
