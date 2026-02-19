package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cubahno/connexions/v2/pkg/config"
	"github.com/redis/go-redis/v9"
	"github.com/sony/gobreaker/v2"
)

// Ensure RedisStore implements gobreaker.SharedDataStore
var _ gobreaker.SharedDataStore = (*RedisStore)(nil)

// RedisStore implements gobreaker.SharedDataStore using Redis.
type RedisStore struct {
	client     *redis.Client
	lockExpiry time.Duration
}

// NewRedisStore creates a new Redis-backed SharedDataStore.
func NewRedisStore(cfg *config.RedisConfig) (*RedisStore, error) {
	if cfg == nil {
		return nil, fmt.Errorf("redis config is nil")
	}

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Address,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisStore{
		client:     client,
		lockExpiry: 10 * time.Second,
	}, nil
}

// Lock acquires a distributed lock for the given name.
func (s *RedisStore) Lock(name string) error {
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
func (s *RedisStore) Unlock(name string) error {
	ctx := context.Background()
	lockKey := s.lockKey(name)

	if err := s.client.Del(ctx, lockKey).Err(); err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}
	return nil
}

// GetData retrieves circuit breaker state data.
func (s *RedisStore) GetData(name string) ([]byte, error) {
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
func (s *RedisStore) SetData(name string, data []byte) error {
	ctx := context.Background()
	dataKey := s.dataKey(name)

	if err := s.client.Set(ctx, dataKey, data, 0).Err(); err != nil {
		return fmt.Errorf("failed to set data: %w", err)
	}
	return nil
}

// Close closes the Redis connection.
func (s *RedisStore) Close() error {
	return s.client.Close()
}

func (s *RedisStore) lockKey(name string) string {
	return fmt.Sprintf("cb:lock:%s", name)
}

func (s *RedisStore) dataKey(name string) string {
	return fmt.Sprintf("cb:data:%s", name)
}
