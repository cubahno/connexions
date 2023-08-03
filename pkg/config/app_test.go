package config

import (
	"testing"
	"time"

	assert2 "github.com/stretchr/testify/assert"
)

func TestNewDefaultAppConfig(t *testing.T) {
	assert := assert2.New(t)

	t.Run("creates default config with correct values", func(t *testing.T) {
		baseDir := "/test/base"
		cfg := NewDefaultAppConfig(baseDir)

		assert.NotNil(cfg)
		assert.Equal("Connexions", cfg.Title)
		assert.Equal(2200, cfg.Port)
		assert.Equal("/.ui", cfg.HomeURL)
		assert.Equal("/.services", cfg.ServiceURL)
		assert.Equal("in-", cfg.ContextAreaPrefix)
		assert.Equal(5*time.Minute, cfg.HistoryDuration)
		assert.False(cfg.DisableUI)

		// Check paths
		assert.NotNil(cfg.Paths)
		assert.Equal(baseDir, cfg.Paths.Base)

		// Check editor config
		assert.NotNil(cfg.Editor)
		assert.Equal("chrome", cfg.Editor.Theme)
		assert.Equal(16, cfg.Editor.FontSize)
	})
}
