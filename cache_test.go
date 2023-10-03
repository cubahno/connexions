package connexions

import (
	"errors"
	"github.com/cubahno/connexions/config"
	"github.com/cubahno/connexions/openapi"
	"github.com/cubahno/connexions/openapi/providers/kin"
	assert2 "github.com/stretchr/testify/assert"
	"net/http"
	"path/filepath"
	"testing"
)

func TestMemoryStorage(t *testing.T) {
	assert := assert2.New(t)
	inst := NewMemoryStorage()

	t.Run("Set", func(t *testing.T) {
		err := inst.Set("key", "value")
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("Get", func(t *testing.T) {
		_ = inst.Set("key", "value")
		res, ok := inst.Get("key")
		assert.True(ok)
		assert.Equal("value", res.(string))
	})

	t.Run("miss", func(t *testing.T) {
		res, ok := inst.Get("new-key")
		assert.False(ok)
		assert.Nil(res)
	})
}

type testFalsyStorage struct {
	MemoryStorage
}

func (s *testFalsyStorage) Get(key string) (any, bool) {
	return nil, false
}

func (s *testFalsyStorage) Set(key string, value any) error {
	return errors.New("cache-set-error")
}

func TestCacheOperationAdapter(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	petStorePath := filepath.Join("testdata", "document-petstore.yml")
	doc, err := kin.NewDocumentFromFile(petStorePath)
	assert.Nil(err)

	addPetOp := doc.FindOperation(&openapi.OperationDescription{Resource: "/pets", Method: http.MethodPost})
	assert.NotNil(addPetOp)

	storage := NewMemoryStorage()
	cachedAddPet := NewCacheOperationAdapter("petstore", addPetOp, storage)

	falsyStorage := &testFalsyStorage{}
	cachedAddPetWithFalsyStorage := NewCacheOperationAdapter("petstore", addPetOp, falsyStorage)

	t.Run("WithParseConfig", func(t *testing.T) {
		res := cachedAddPet.WithParseConfig(&config.ParseConfig{MaxLevels: 2})
		assert.Equal(cachedAddPet, res)
	})

	t.Run("ID", func(t *testing.T) {
		assert.Equal("addPet", cachedAddPet.ID())
	})

	t.Run("GetParameters", func(t *testing.T) {
		_ = cachedAddPet.GetParameters()
		res := cachedAddPet.GetParameters()
		assert.Equal(addPetOp.GetParameters(), res)

		c, ok := storage.Get("petstore:addPet:parameters")
		assert.True(ok)
		assert.Equal(res, c)
	})

	t.Run("GetParameters-missed", func(t *testing.T) {
		res := cachedAddPetWithFalsyStorage.GetParameters()
		assert.Equal(addPetOp.GetParameters(), res)
	})

	t.Run("GetRequestBody", func(t *testing.T) {
		opSchema, opContentType := addPetOp.GetRequestBody()
		_, _ = cachedAddPet.GetRequestBody()
		schema, contentType := cachedAddPet.GetRequestBody()

		assert.Equal(opSchema, schema)
		assert.Equal(opContentType, contentType)

		c, ok := storage.Get("petstore:addPet:requestBody")
		assert.True(ok)
		assert.Equal(&SchemaWithContentType{
			Schema:      opSchema,
			ContentType: opContentType,
		}, c)
	})

	t.Run("GetRequestBody-missed", func(t *testing.T) {
		opSchema, opContentType := addPetOp.GetRequestBody()
		schema, contentType := cachedAddPetWithFalsyStorage.GetRequestBody()

		assert.Equal(opSchema, schema)
		assert.Equal(opContentType, contentType)
	})

	t.Run("GetResponse", func(t *testing.T) {
		_ = cachedAddPet.GetResponse()
		res := cachedAddPet.GetResponse()
		assert.Equal(addPetOp.GetResponse(), res)

		c, ok := storage.Get("petstore:addPet:response")
		assert.True(ok)
		assert.Equal(res, c)
	})

	t.Run("GetResponse-missed", func(t *testing.T) {
		res := cachedAddPetWithFalsyStorage.GetResponse()
		assert.Equal(addPetOp.GetResponse(), res)
	})
}
