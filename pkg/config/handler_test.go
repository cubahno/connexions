package config

import (
	"testing"

	assert2 "github.com/stretchr/testify/assert"
)

func TestNewHandlerConfig(t *testing.T) {
	assert := assert2.New(t)

	t.Run("creates handler config from service config", func(t *testing.T) {
		validateCfg := &ValidateConfig{
			Request:  true,
			Response: true,
		}
		serviceCfg := &ServiceConfig{
			Validate: validateCfg,
		}

		handlerCfg := NewHandlerConfig(serviceCfg)

		assert.NotNil(handlerCfg)
		assert.Equal(validateCfg, handlerCfg.Validate)
	})

	t.Run("handles nil validate config", func(t *testing.T) {
		serviceCfg := &ServiceConfig{
			Validate: nil,
		}

		handlerCfg := NewHandlerConfig(serviceCfg)

		assert.NotNil(handlerCfg)
		assert.Nil(handlerCfg.Validate)
	})
}
