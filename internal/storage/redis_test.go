package storage

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/cubahno/connexions/v2/pkg/config"
	assert2 "github.com/stretchr/testify/assert"
)

func TestNewRedisStore(t *testing.T) {
	assert := assert2.New(t)

	t.Run("nil config returns error", func(t *testing.T) {
		store, err := NewRedisStore(nil)
		assert.Nil(store)
		assert.Error(err)
		assert.Contains(err.Error(), "redis config is nil")
	})

	t.Run("invalid address returns error", func(t *testing.T) {
		cfg := &config.RedisConfig{
			Address:  "invalid:99999",
			Password: "",
			DB:       0,
		}
		store, err := NewRedisStore(cfg)
		assert.Nil(store)
		assert.Error(err)
		assert.Contains(err.Error(), "failed to connect to redis")
	})

	t.Run("successful connection", func(t *testing.T) {
		mr := miniredis.RunT(t)
		cfg := &config.RedisConfig{
			Address: mr.Addr(),
		}
		store, err := NewRedisStore(cfg)
		assert.NoError(err)
		assert.NotNil(store)
		_ = store.Close()
	})
}

func TestRedisStore_KeyFormats(t *testing.T) {
	assert := assert2.New(t)

	store := &RedisStore{}

	t.Run("lock key format", func(t *testing.T) {
		key := store.lockKey("test-breaker")
		assert.Equal("cb:lock:test-breaker", key)
	})

	t.Run("data key format", func(t *testing.T) {
		key := store.dataKey("test-breaker")
		assert.Equal("cb:data:test-breaker", key)
	})
}

func TestRedisStore_Lock(t *testing.T) {
	assert := assert2.New(t)
	mr := miniredis.RunT(t)

	store, err := NewRedisStore(&config.RedisConfig{Address: mr.Addr()})
	assert.NoError(err)
	defer func() { _ = store.Close() }()

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

func TestRedisStore_Unlock(t *testing.T) {
	assert := assert2.New(t)
	mr := miniredis.RunT(t)

	store, err := NewRedisStore(&config.RedisConfig{Address: mr.Addr()})
	assert.NoError(err)
	defer func() { _ = store.Close() }()

	t.Run("unlock existing lock", func(t *testing.T) {
		name := "test-unlock"
		_ = store.Lock(name)

		err := store.Unlock(name)
		assert.NoError(err)
	})

	t.Run("unlock non-existent lock", func(t *testing.T) {
		err := store.Unlock("non-existent")
		assert.NoError(err) // DEL returns 0 but no error
	})
}

func TestRedisStore_GetData(t *testing.T) {
	assert := assert2.New(t)
	mr := miniredis.RunT(t)

	store, err := NewRedisStore(&config.RedisConfig{Address: mr.Addr()})
	assert.NoError(err)
	defer func() { _ = store.Close() }()

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

func TestRedisStore_SetData(t *testing.T) {
	assert := assert2.New(t)
	mr := miniredis.RunT(t)

	store, err := NewRedisStore(&config.RedisConfig{Address: mr.Addr()})
	assert.NoError(err)
	defer func() { _ = store.Close() }()

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

func TestRedisStore_Close(t *testing.T) {
	assert := assert2.New(t)
	mr := miniredis.RunT(t)

	store, err := NewRedisStore(&config.RedisConfig{Address: mr.Addr()})
	assert.NoError(err)

	err = store.Close()
	assert.NoError(err)
}
