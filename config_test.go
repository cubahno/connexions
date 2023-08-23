package connexions

import (
	"fmt"
	assert2 "github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAppConfig(t *testing.T) {
	assert := assert2.New(t)

	for _, tc := range []struct {
		prefix  string
		isValid bool
	}{
		{"/", true},
		{"/api", true},
		{"/.services", false},
		{"/.settings", false},
		{"/.contexts", false},
		{"/.ui", false},
	} {
		t.Run(fmt.Sprintf("IsValidPrefix: %s", tc.prefix), func(t *testing.T) {
			cfg := NewDefaultConfig().App
			assert.True(cfg.IsValidPrefix(tc.prefix) == tc.isValid)
		})
	}
}

func TestConfig(t *testing.T) {
	assert := assert2.New(t)

	t.Run("GetServiceConfig", func(t *testing.T) {
		cfg := &Config{}
		cfg.Services = map[string]*ServiceConfig{
			"service1": {
				Latency: 100 * time.Millisecond,
			},
		}
		res := cfg.GetServiceConfig("service1")
		assert.Equal(100*time.Millisecond, res.Latency)
		assert.Equal(&ServiceError{}, res.Errors)
		assert.Equal(&ServiceValidateConfig{
			Request:  true,
			Response: false,
		}, res.Validate)
	})

	t.Run("GetServiceConfig-Default", func(t *testing.T) {
		cfg := &Config{}
		cfg.Services = map[string]*ServiceConfig{
			"service1": {
				Latency: 100 * time.Millisecond,
			},
		}
		res := cfg.GetServiceConfig("service-2")
		assert.Equal(0*time.Millisecond, res.Latency)
		assert.Equal(&ServiceError{}, res.Errors)
		assert.Equal(&ServiceValidateConfig{
			Request:  true,
			Response: false,
		}, res.Validate)
	})

	t.Run("EnsureConfigValues-when-empty", func(t *testing.T) {
		cfg := &Config{}
		cfg.EnsureConfigValues()
		assert.Equal(NewDefaultConfig(), cfg)
	})

	t.Run("EnsureConfigValues-port-is-set", func(t *testing.T) {
		cfg := &Config{
			App: &AppConfig{},
		}
		cfg.EnsureConfigValues()

		expected := NewDefaultConfig()
		expected.App.ServeUI = false
		expected.App.ServeSpec = false
		assert.Equal(expected, cfg)
	})

	t.Run("EnsureConfigValues-when-partial-app", func(t *testing.T) {
		cfg := &Config{
			App: &AppConfig{
				Port: 5555,
			},
		}
		cfg.EnsureConfigValues()

		expected := NewDefaultConfig()
		expected.App.Port = 5555
		expected.App.ServeUI = false
		expected.App.ServeSpec = false
		assert.Equal(expected, cfg)
	})
}

func TestServiceError(t *testing.T) {
	assert := assert2.New(t)

	t.Run("GetError-with-100%-chance", func(t *testing.T) {
		err := &ServiceError{Chance: 100}
		assert.Equal(500, err.GetError())
	})

	t.Run("GetError-with-0-chance", func(t *testing.T) {
		err := &ServiceError{Chance: 0}
		assert.Equal(0, err.GetError())
	})

	t.Run("GetError-with-100%-chance-and-100%-code", func(t *testing.T) {
		err := &ServiceError{
			Chance: 100,
			Codes:  map[int]int{429: 100},
		}
		assert.Equal(429, err.GetError())
	})

	t.Run("GetError-with-100%-chance-and-single-10%-code", func(t *testing.T) {
		err := &ServiceError{
			Chance: 100,
			Codes:  map[int]int{429: 10},
		}
		assert.Contains([]int{429, 500}, err.GetError())
	})

	t.Run("GetError-with-100%-chance-and-50-50-no-default-codes", func(t *testing.T) {
		err := &ServiceError{
			Chance: 100,
			Codes:  map[int]int{400: 50, 429: 500},
		}
		assert.Contains([]int{400, 429}, err.GetError())
	})

	t.Run("GetError-with-50%-chance-and-50-50-no-default-codes", func(t *testing.T) {
		err := &ServiceError{
			Chance: 100,
			Codes:  map[int]int{400: 50, 429: 500},
		}
		assert.Contains([]int{0, 400, 429}, err.GetError())
	})

	t.Run("GetError-returns-default", func(t *testing.T) {
		err := &ServiceError{
			Chance: 100,
			Codes:  map[int]int{},
		}
		assert.Equal(500, err.GetError())
	})
}

func TestNewConfigFromFile(t *testing.T) {
	assert := assert2.New(t)

	t.Run("happy-path", func(t *testing.T) {
		tempDir := t.TempDir()
		contents := `
app:
  port: 8000
  homeUrl: /new-ui
  serviceUrl: /new-services
  contextUrl: /new-contexts
  settingsUrl: /new-settings
  serveUI: true
  serveSpec: true
  contextAreaPrefix: from-

services:
  foo:
    latency: 1.25s
    errors:
      chance: 50%
      codes:
        400: 51%
        500: 52
    contexts:
      - petstore:
      - fake: pet
      - fake: gamer
    validate:
      request: true
      response: true
`
		expected := &Config{
			App: &AppConfig{
				Port:              8000,
				HomeURL:           "/new-ui",
				ServiceURL:        "/new-services",
				ContextURL:        "/new-contexts",
				SettingsURL:       "/new-settings",
				ServeUI:           true,
				ServeSpec:         true,
				ContextAreaPrefix: "from-",
			},
			Services: map[string]*ServiceConfig{
				"foo": {
					Latency: 1250 * time.Millisecond,
					Errors: &ServiceError{
						Chance: 50,
						Codes: map[int]int{
							400: 51,
							500: 52,
						},
					},
					Contexts: []map[string]string{
						{"petstore": ""},
						{"fake": "pet"},
						{"fake": "gamer"},
					},
					Validate: &ServiceValidateConfig{
						Request:  true,
						Response: true,
					},
				},
			},
		}

		filePath := filepath.Join(tempDir, "cfg.yml")
		err := os.WriteFile(filePath, []byte(contents), 0644)
		assert.Nil(err)

		cfg, err := NewConfigFromFile(filePath)
		assert.Nil(err)
		assert.Equal(expected, cfg)

		// check live config is triggered
		ymlContent, _ := yaml.Marshal(cfg)
		yamlStr := string(ymlContent)
		yamlStr = strings.Replace(yamlStr, "port: 8000", "port: 9000", 1)
		_ = os.WriteFile(filePath, []byte(yamlStr), 0644)

		// we need a moment there
		time.Sleep(100 * time.Millisecond)
		app := cfg.GetApp()
		assert.Equal(9000, app.Port)
	})

	t.Run("invalid-type-update-dont-kill", func(t *testing.T) {
		tempDir := t.TempDir()
		contents := `
app:
  port: 8000
`
		filePath := filepath.Join(tempDir, "cfg.yml")
		err := os.WriteFile(filePath, []byte(contents), 0644)
		assert.Nil(err)

		cfg, err := NewConfigFromFile(filePath)
		assert.Nil(err)

		// check live config is triggered
		ymlContent, _ := yaml.Marshal(cfg)
		yamlStr := string(ymlContent)
		// set invalid type
		yamlStr = strings.Replace(yamlStr, "port: 8000", "port: x", 1)
		_ = os.WriteFile(filePath, []byte(yamlStr), 0644)

		// we need a moment there
		time.Sleep(50 * time.Millisecond)
		// port is still the old one
		assert.Equal(8000, cfg.App.Port)
	})

	t.Run("invalid-yaml-transform-dont-kill", func(t *testing.T) {
		tempDir := t.TempDir()
		contents := `
app:
  port: 8000
services:
  foo:
    latency: 1.25s
    errors:
      chance: 50%
`
		filePath := filepath.Join(tempDir, "cfg.yml")
		err := os.WriteFile(filePath, []byte(contents), 0644)
		assert.Nil(err)

		cfg, err := NewConfigFromFile(filePath)
		assert.Nil(err)

		// check live config is triggered
		ymlContent, _ := yaml.Marshal(cfg)
		yamlStr := string(ymlContent)
		// set invalid type
		yamlStrBad := strings.Replace(yamlStr, "chance: 50", "chance: x%", 1)
		_ = os.WriteFile(filePath, []byte(yamlStrBad), 0644)

		// we need a moment there
		time.Sleep(50 * time.Millisecond)
		// port is still the old one
		app := cfg.GetApp()
		assert.Equal(8000, app.Port)

		// invalid yaml written
		yamlStrBad = strings.Replace(yamlStr, "chance: 50", "1", 1)
		_ = os.WriteFile(filePath, []byte(yamlStrBad), 0644)

		// port is still the old one
		app = cfg.GetApp()
		assert.Equal(8000, app.Port)
	})

	t.Run("invalid-yaml", func(t *testing.T) {
		tempDir := t.TempDir()
		contents := `1`
		filePath := filepath.Join(tempDir, "cfg.yml")
		err := os.WriteFile(filePath, []byte(contents), 0644)
		assert.Nil(err)

		cfg, err := NewConfigFromFile(filePath)
		assert.Nil(cfg)
		assert.NotNil(err)
	})

	t.Run("transformation-error", func(t *testing.T) {
		tempDir := t.TempDir()
		contents := `
app:
  port: xxx
`
		filePath := filepath.Join(tempDir, "cfg.yml")
		err := os.WriteFile(filePath, []byte(contents), 0644)
		assert.Nil(err)

		cfg, err := NewConfigFromFile(filePath)
		assert.Nil(cfg)
		assert.NotNil(err)
	})

	t.Run("file-not-found", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "cfg.yml")

		cfg, err := NewConfigFromFile(filePath)
		assert.Nil(cfg)
		assert.NotNil(err)
	})
}

func TestNewConfigFromContent(t *testing.T) {
	assert := assert2.New(t)

	t.Run("happy-path", func(t *testing.T) {
		contents := `
app:
  port: 8000
  homeUrl: /new-ui
  serviceUrl: /new-services
  contextUrl: /new-contexts
  settingsUrl: /new-settings
  serveUI: true
  serveSpec: true
  contextAreaPrefix: from-
`
		expected := &Config{
			App: &AppConfig{
				Port:              8000,
				HomeURL:           "/new-ui",
				ServiceURL:        "/new-services",
				ContextURL:        "/new-contexts",
				SettingsURL:       "/new-settings",
				ServeUI:           true,
				ServeSpec:         true,
				ContextAreaPrefix: "from-",
			},
		}

		cfg, err := NewConfigFromContent([]byte(contents))
		assert.Nil(err)
		assert.Equal(expected, cfg)
	})

	t.Run("invalid-yaml-properties", func(t *testing.T) {
		contents := `root:\nfoo:  bar`
		cfg, err := NewConfigFromContent([]byte(contents))
		assert.Nil(cfg)
		assert.Equal(ErrInvalidConfig, err)
	})

	t.Run("invalid-yaml", func(t *testing.T) {
		contents := `1`
		cfg, err := NewConfigFromContent([]byte(contents))
		assert.Nil(cfg)
		assert.NotNil(err)
	})

	t.Run("transformation-error", func(t *testing.T) {
		contents := `
app:
  port: xxx
`
		cfg, err := NewConfigFromContent([]byte(contents))
		assert.Nil(cfg)
		assert.NotNil(err)
	})

}

func TestNewDefaultConfig(t *testing.T) {
	assert := assert2.New(t)

	cfg := NewDefaultConfig()
	expected := &Config{
		App: &AppConfig{
			Port:              2200,
			HomeURL:           "/.ui",
			ServiceURL:        "/.services",
			SettingsURL:       "/.settings",
			ContextURL:        "/.contexts",
			ServeUI:           true,
			ServeSpec:         true,
			ContextAreaPrefix: "in-",
		},
	}
	assert.Equal(expected, cfg)
}
