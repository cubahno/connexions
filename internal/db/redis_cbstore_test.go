package db

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/sony/gobreaker/v2"
	assert2 "github.com/stretchr/testify/assert"
)

func TestRedisCircuitBreakerStore_ImplementsInterface(t *testing.T) {
	var _ gobreaker.SharedDataStore = (*redisCircuitBreakerStore)(nil)
}

func TestRedisCircuitBreakerStore_KeyFormats(t *testing.T) {
	assert := assert2.New(t)
	store := &redisCircuitBreakerStore{}

	t.Run("lock key format", func(t *testing.T) {
		key := store.lockKey("test-breaker")
		assert.Equal("cb:lock:test-breaker", key)
	})

	t.Run("data key format", func(t *testing.T) {
		key := store.dataKey("test-breaker")
		assert.Equal("cb:data:test-breaker", key)
	})
}

func TestRedisCircuitBreakerStore_Lock(t *testing.T) {
	assert := assert2.New(t)
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = client.Close() }()

	store := newRedisCircuitBreakerStore(client)

	t.Run("acquire lock", func(t *testing.T) {
		err := store.Lock("test-lock")
		assert.NoError(err)
	})

	t.Run("lock already held", func(t *testing.T) {
		name := "test-lock-2"
		err := store.Lock(name)
		assert.NoError(err)

		err = store.Lock(name)
		assert.Error(err)
		assert.Contains(err.Error(), "lock already held")
	})

	t.Run("reacquire after unlock", func(t *testing.T) {
		name := "test-lock-3"
		err := store.Lock(name)
		assert.NoError(err)

		err = store.Unlock(name)
		assert.NoError(err)

		err = store.Lock(name)
		assert.NoError(err)
	})
}

func TestRedisCircuitBreakerStore_Unlock(t *testing.T) {
	assert := assert2.New(t)
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = client.Close() }()

	store := newRedisCircuitBreakerStore(client)

	t.Run("unlock existing lock", func(t *testing.T) {
		name := "test-unlock"
		_ = store.Lock(name)

		err := store.Unlock(name)
		assert.NoError(err)
	})

	t.Run("unlock non-existent lock", func(t *testing.T) {
		err := store.Unlock("non-existent")
		assert.NoError(err)
	})
}

func TestRedisCircuitBreakerStore_GetData(t *testing.T) {
	assert := assert2.New(t)
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = client.Close() }()

	store := newRedisCircuitBreakerStore(client)

	t.Run("get non-existent data returns nil", func(t *testing.T) {
		data, err := store.GetData("non-existent")
		assert.NoError(err)
		assert.Nil(data)
	})

	t.Run("get existing data", func(t *testing.T) {
		name := "test-get"
		testData := []byte(`{"state":"closed"}`)
		_ = store.SetData(name, testData)

		data, err := store.GetData(name)
		assert.NoError(err)
		assert.Equal(testData, data)
	})
}

func TestRedisCircuitBreakerStore_SetData(t *testing.T) {
	assert := assert2.New(t)
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = client.Close() }()

	store := newRedisCircuitBreakerStore(client)

	t.Run("set data", func(t *testing.T) {
		testData := []byte(`{"state":"open","counts":{"requests":5}}`)
		err := store.SetData("test-set", testData)
		assert.NoError(err)

		data, _ := store.GetData("test-set")
		assert.Equal(testData, data)
	})

	t.Run("overwrite existing data", func(t *testing.T) {
		name := "test-overwrite"
		_ = store.SetData(name, []byte("old"))

		newData := []byte("new")
		err := store.SetData(name, newData)
		assert.NoError(err)

		data, _ := store.GetData(name)
		assert.Equal(newData, data)
	})
}
