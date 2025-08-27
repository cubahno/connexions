package config

import (
	"log"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/cubahno/connexions/internal/types"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/rawbytes"
)

type KeyValue[K, V any] struct {
	Key   K
	Value V
}

// Config is the main configuration struct.
// App is the app config.
// Services is a map of service name and the corresponding config.
// ServiceName is the first part of the path.
// e.g. /petstore/v1/pets -> petstore
// in case, there's no service name, the name "root" will be used.
type Config struct {
	App      *AppConfig                `koanf:"app" yaml:"app"`
	Services map[string]*ServiceConfig `koanf:"services" yaml:"services"`
	BaseDir  string                    `koanf:"-"`
	mu       sync.Mutex
}

const (
	// RootServiceName is the name and location in the service directory of the service without a name.
	RootServiceName = "root"

	// RootOpenAPIName is the name and location of the OpenAPI service without a name.
	RootOpenAPIName = "openapi"
)

type EditorConfig struct {
	Theme    string `koanf:"theme" json:"theme" yaml:"theme"`
	FontSize int    `koanf:"fontSize" json:"fontSize" yaml:"fontSize"`
}

func (c *Config) GetApp() *AppConfig {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.App
}

// GetServiceConfig returns the config for a service.
// If the service is not found, it returns a default config.
func (c *Config) GetServiceConfig(service string) *ServiceConfig {
	c.mu.Lock()
	defer c.mu.Unlock()

	res, ok := c.Services[service]
	if !ok || res == nil {
		res = NewServiceConfig()
	}

	if len(res.errors) == 0 && res.Errors != nil {
		res.errors = make([]*KeyValue[int, int], 0)

	}

	if res.Validate == nil {
		res.Validate = NewServiceValidateConfig()
	}

	if res.Cache == nil {
		res.Cache = NewServiceCacheConfig()
	}

	if res.latencies == nil && res.Latencies != nil {
		res.latencies = res.ParseLatencies()
	}

	return res
}

// EnsureConfigValues ensures that all config values are set.
func (c *Config) EnsureConfigValues() {
	defaultConfig := NewDefaultConfig(c.BaseDir)
	app := c.GetApp()

	c.mu.Lock()
	defer c.mu.Unlock()

	if app == nil {
		app = defaultConfig.App
	}

	if c.Services == nil {
		c.Services = defaultConfig.Services
	}

	if app.Port == 0 {
		app.Port = defaultConfig.App.Port
	}
	if app.HomeURL == "" {
		app.HomeURL = defaultConfig.App.HomeURL
	}
	if app.ServiceURL == "" {
		app.ServiceURL = defaultConfig.App.ServiceURL
	}
	if app.SettingsURL == "" {
		app.SettingsURL = defaultConfig.App.SettingsURL
	}
	if app.ContextURL == "" {
		app.ContextURL = defaultConfig.App.ContextURL
	}
	if app.ContextAreaPrefix == "" {
		app.ContextAreaPrefix = defaultConfig.App.ContextAreaPrefix
	}
	if app.Editor == nil {
		app.Editor = defaultConfig.App.Editor
	}
	if app.HistoryDuration == 0 {
		app.HistoryDuration = defaultConfig.App.HistoryDuration
	}

	app.Paths = defaultConfig.App.Paths
	c.App = app
}

// transformConfig applies transformations to the config.
// Currently, it removes % from the chances.
func (c *Config) transformConfig(k *koanf.Koanf) *koanf.Koanf {
	transformed := koanf.New(".")
	for key, value := range k.All() {
		envKey := strings.ToUpper(types.ToSnakeCase(key))
		finalValue := value

		switch v := value.(type) {
		case int, float64, bool:
			if envValue, exists := os.LookupEnv(envKey); exists {
				finalValue = envValue
			}
		case string:
			// Check if the value is a string and ends with '%'
			if strings.HasSuffix(v, "%") {
				finalValue = strings.TrimSuffix(v, "%")
			}

			// Check for corresponding environment variable
			if envValue, exists := os.LookupEnv(envKey); exists {
				finalValue = envValue
			}
		}

		_ = transformed.Set(key, finalValue)
	}
	return transformed
}

func (c *Config) Reload() {
	c.mu.Lock()
	defer c.mu.Unlock()

	filePath := c.App.Paths.ConfigFile
	provider := file.Provider(filePath)

	// Throw away the old config and load a fresh copy.
	log.Println("reloading config ...")
	k := koanf.New(".")
	if err := k.Load(provider, yaml.Parser()); err != nil {
		slog.Error("error loading config", "error", err)
		return
	}

	transformed := c.transformConfig(k)
	if err := transformed.Unmarshal("", c); err != nil {
		slog.Error("error unmarshalling config", "error", err)
		return
	}

	slog.Info("Configuration reloaded!")
	slog.Info(k.Sprint())
}

// MustConfig creates a new config from a YAML file path.
// In case it file does not exist or has incorrect YAML:
// - it creates a new default config
//
// Koanf has a file watcher, but its easier to control the changes with a manual reload.
func MustConfig(baseDir string) *Config {
	paths := NewPaths(baseDir)
	filePath := paths.ConfigFile

	res := NewDefaultConfig(baseDir)

	k := koanf.New(".")
	provider := file.Provider(filePath)
	if err := k.Load(provider, yaml.Parser()); err != nil {
		slog.Error("error loading config. using fallback", "error", err)
		return res
	}

	cfg := res
	transformed := cfg.transformConfig(k)
	if err := transformed.Unmarshal("", cfg); err != nil {
		slog.Error("error loading config. using fallback", "error", err)
		return res
	}
	cfg.EnsureConfigValues()
	cfg.App.Paths = paths
	cfg.BaseDir = baseDir

	return cfg
}

// NewConfigFromContent creates a new config from a YAML file content.
func NewConfigFromContent(content []byte) (*Config, error) {
	k := koanf.New(".")
	provider := rawbytes.Provider(content)
	if err := k.Load(provider, yaml.Parser()); err != nil {
		return nil, err
	}

	cfg := &Config{}
	transformed := cfg.transformConfig(k)
	if err := transformed.Unmarshal("", cfg); err != nil {
		return nil, err
	}

	cfg.EnsureConfigValues()

	return cfg, nil
}

// NewDefaultConfig creates a new default config in case the config file is missing, not found or any other error.
func NewDefaultConfig(baseDir string) *Config {
	return &Config{
		App:      NewDefaultAppConfig(baseDir),
		Services: make(map[string]*ServiceConfig),
		BaseDir:  baseDir,
	}
}
