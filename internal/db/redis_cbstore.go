package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sony/gobreaker/v2"
)

// Ensure redisCircuitBreakerStore implements gobreaker.SharedDataStore
var _ gobreaker.SharedDataStore = (*redisCircuitBreakerStore)(nil)

// redisCircuitBreakerStore is a Redis-backed implementation of gobreaker.SharedDataStore.
type redisCircuitBreakerStore struct {
	client     *redis.Client
	lockExpiry time.Duration
}

// newRedisCircuitBreakerStore creates a new Redis-backed circuit breaker store.
func newRedisCircuitBreakerStore(client *redis.Client) *redisCircuitBreakerStore {
	return &redisCircuitBreakerStore{
		client:     client,
		lockExpiry: 10 * time.Second,
	}
}

// Lock acquires a distributed lock for the given name.
func (s *redisCircuitBreakerStore) Lock(name string) error {
	ctx := context.Background()
	lockKey := s.lockKey(name)

	// Try to acquire lock with SET NX (only set if not exists)
	result, err := s.client.SetArgs(ctx, lockKey, "1", redis.SetArgs{
		Mode: "NX",
		TTL:  s.lockExpiry,
	}).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	if result == "" {
		return fmt.Errorf("lock already held for %s", name)
	}
	return nil
}

// Unlock releases the distributed lock for the given name.
func (s *redisCircuitBreakerStore) Unlock(name string) error {
	ctx := context.Background()
	lockKey := s.lockKey(name)

	if err := s.client.Del(ctx, lockKey).Err(); err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}
	return nil
}

// GetData retrieves circuit breaker state data.
func (s *redisCircuitBreakerStore) GetData(name string) ([]byte, error) {
	ctx := context.Background()
	dataKey := s.dataKey(name)

	data, err := s.client.Get(ctx, dataKey).Bytes()
	if errors.Is(err, redis.Nil) {
		// No data yet
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get data: %w", err)
	}
	return data, nil
}

// SetData stores circuit breaker state data.
func (s *redisCircuitBreakerStore) SetData(name string, data []byte) error {
	ctx := context.Background()
	dataKey := s.dataKey(name)

	if err := s.client.Set(ctx, dataKey, data, 0).Err(); err != nil {
		return fmt.Errorf("failed to set data: %w", err)
	}
	return nil
}

func (s *redisCircuitBreakerStore) lockKey(name string) string {
	return fmt.Sprintf("cb:lock:%s", name)
}

func (s *redisCircuitBreakerStore) dataKey(name string) string {
	return fmt.Sprintf("cb:data:%s", name)
}

