package connexions

import (
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/rawbytes"
	"log"
	"math/rand"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Config struct {
	// App is the app config.
	App *AppConfig `koanf:"app"`

	// Services is a map of service name and the corresponding config.
	// ServiceName is the first part of the path.
	// e.g. /petstore/v1/pets -> petstore
	// in case, there's no service name, the name ".root" will be used.
	Services map[string]*ServiceConfig `koanf:"services"`

	baseDir string `koanf:"-"`
	mu      sync.Mutex
}

type ServiceConfig struct {
	// Latency is the latency to add to the response.
	// Latency not used in the services API, only when endpoint queried directly.
	Latency time.Duration `koanf:"latency"`

	// Errors is the error config.
	Errors *ServiceError `koanf:"errors"`

	// Contexts is the list of contexts to use for replacements.
	// It is a map of context name defined either in the UI or filename without extension.
	// You can refer to the name when building aliases.
	Contexts []map[string]string `koanf:"contexts"`

	// ParseConfig is the config for parsing the OpenAPI spec.
	ParseConfig *ParseConfig `json:"parseConfig" koanf:"parseConfig"`

	// Validate is the validation config.
	// It is used to validate the request and/or response outside of the Services API.
	Validate *ServiceValidateConfig `koanf:"validate"`

	// Cache is the cache config.
	Cache *ServiceCacheConfig `koanf:"cache"`
}

type ServiceError struct {
	// Chance is the chance to return an error.
	// In the config, it can be set with %-suffix.
	Chance int `koanf:"chance"`

	// Codes is a map of error codes and their weights if Chance > 0.
	// If no error codes are specified, it returns a 500 error code.
	Codes map[int]int `koanf:"codes"`

	mu sync.Mutex
}

type ServiceValidateConfig struct {
	// Request is a flag whether to validate the request.
	// Default: true
	Request bool `koanf:"request"`

	// Response is a flag whether to validate the response.
	// Default: false
	Response bool `koanf:"response"`
}

type ServiceCacheConfig struct {
	// Avoid multiple schema parsing by caching the parsed schema.
	// Default: true
	Schema bool `koanf:"schema"`
}

type ParseConfig struct {
	// MaxLevels is the maximum level to parse.
	MaxLevels int `koanf:"maxLevels"`

	// MaxRecursionLevels is the maximum level to parse recursively.
	// 0 means no recursion: property will get nil value.
	MaxRecursionLevels int `koanf:"maxRecursionLevels"`

	// OnlyRequired is a flag whether to include only required fields.
	// If the spec contains deep references, this might significantly speed up parsing.
	OnlyRequired bool `koanf:"onlyRequired"`
}

const (
	// RootServiceName is the name and location in the service directory of the service without a name.
	RootServiceName = ".root"

	// RootOpenAPIName is the name and location of the OpenAPI service without a name.
	RootOpenAPIName = ".openapi"
)

type SchemaProvider string

const (
	KinOpenAPIProvider    SchemaProvider = "kin-openapi"
	LibOpenAPIProvider    SchemaProvider = "libopenapi"
	DefaultSchemaProvider SchemaProvider = LibOpenAPIProvider
)

// AppConfig is the app configuration.
type AppConfig struct {
	// Port is the port number to listen on.
	Port int `json:"port" koanf:"port"`

	// HomeURL is the URL for the UI home page.
	HomeURL string `json:"homeUrl" koanf:"homeUrl"`

	// ServiceURL is the URL for the service and resources endpoints in the UI.
	ServiceURL string `json:"serviceUrl" koanf:"serviceUrl"`

	// SettingsURL is the URL for the settings endpoint in the UI.
	SettingsURL string `json:"settingsUrl" koanf:"settingsUrl"`

	// ContextURL is the URL for the context endpoint in the UI.
	ContextURL string `json:"contextUrl" koanf:"contextUrl"`

	// ContextAreaPrefix sets sub-contexts for replacements in path, header or any other supported place.
	//
	// for example:
	// in-path:
	//   user_id: "fake.ids.int8"
	ContextAreaPrefix string `json:"contextAreaPrefix" koanf:"contextAreaPrefix"`

	// DisableUI is a flag whether to disable the UI.
	DisableUI bool `json:"disableUI" koanf:"disableUI"`

	// DisableSpec is a flag whether to disable the Swagger UI.
	DisableSwaggerUI bool `json:"disableSwaggerUI" koanf:"disableSwaggerUI"`

	// SchemaProvider is the schema provider to use: kin-openapi or libopenapi.
	SchemaProvider SchemaProvider `json:"schemaProvider" koanf:"schemaProvider"`

	// Paths is the paths to various resource directories.
	Paths *Paths `json:"-" koanf:"-"`

	// CreateFileStructure is a flag whether to create the initial resources file structure:
	// contexts, services, etc.
	// It will also copy sample files from the samples directory into services.
	// Default: true in config.dist.yml which gets initially copied to config.yml.
	CreateFileStructure bool `koanf:"createFileStructure"`
}

func NewPaths(baseDir string) *Paths {
	resDir := filepath.Join(baseDir, "resources")
	dataDir := filepath.Join(resDir, "data")
	svcDir := filepath.Join(dataDir, "services")

	return &Paths{
		Base:      baseDir,
		Resources: resDir,

		Data:              dataDir,
		Contexts:          filepath.Join(dataDir, "contexts"),
		ConfigFile:        filepath.Join(dataDir, "config.yml"),
		Services:          svcDir,
		ServicesOpenAPI:   filepath.Join(svcDir, RootOpenAPIName),
		ServicesFixedRoot: filepath.Join(svcDir, RootServiceName),

		Docs:    filepath.Join(resDir, "docs"),
		Samples: filepath.Join(resDir, "samples"),
		UI:      filepath.Join(resDir, "ui"),
	}
}

// IsValidPrefix returns true if the prefix is not a reserved URL.
func (a *AppConfig) IsValidPrefix(prefix string) bool {
	return !SliceContains([]string{
		a.HomeURL,
		a.ServiceURL,
		a.SettingsURL,
		a.ContextURL,
	}, prefix)
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
	if !ok {
		res = &ServiceConfig{}
	}

	if res.Errors == nil {
		res.Errors = &ServiceError{}
	}

	if res.Validate == nil {
		res.Validate = &ServiceValidateConfig{
			Request:  true,
			Response: false,
		}
	}

	if res.Cache == nil {
		res.Cache = &ServiceCacheConfig{
			Schema: true,
		}
	}

	return res
}

// EnsureConfigValues ensures that all config values are set.
func (c *Config) EnsureConfigValues() {
	defaultConfig := NewDefaultConfig(c.baseDir)
	app := c.GetApp()

	c.mu.Lock()
	defer c.mu.Unlock()

	if app == nil {
		c.App = defaultConfig.App
		return
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
	if app.SchemaProvider == "" {
		app.SchemaProvider = defaultConfig.App.SchemaProvider
	}

	app.Paths = defaultConfig.App.Paths
	c.App = app
}

// transformConfig applies transformations to the config.
// Currently, it removes % from the chances.
func (c *Config) transformConfig(k *koanf.Koanf) *koanf.Koanf {
	transformed := koanf.New(".")
	for key, value := range k.All() {
		if v, isString := value.(string); isString && strings.HasSuffix(v, "%") {
			value = strings.TrimSuffix(v, "%")
		}
		_ = transformed.Set(key, value)
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
		log.Printf("error loading config: %v\n", err)
		return
	}

	transformed := c.transformConfig(k)
	if err := transformed.Unmarshal("", c); err != nil {
		log.Printf("error unmarshalling config: %v\n", err)
		return
	}

	log.Println("Configuration reloaded!")
	log.Println(k.Sprint())
}

// GetError returns an error code based on the chance and error weights.
// If no error weights are specified, it returns a 500 error code.
// If the chance is 0, it returns 0.
func (s *ServiceError) GetError() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	randomNumber := rand.Intn(100) + 1
	if randomNumber > s.Chance {
		return 0
	}

	defaultErrorCode := 500
	errorWeights := s.Codes
	// If no error weights are specified, return a 500 error code
	if errorWeights == nil {
		return defaultErrorCode
	}

	// Calculate the total weight
	totalWeight := 0
	for _, weight := range errorWeights {
		totalWeight += weight
	}

	// Generate a random number between 1 and totalWeight
	if totalWeight > 0 {
		randomNumber = rand.Intn(totalWeight) + 1
	}

	// Select an error code based on the random number and weights
	for code, weight := range errorWeights {
		randomNumber -= weight
		if randomNumber <= 0 {
			return code
		}
	}

	return defaultErrorCode
}

// MustConfig creates a new config from a YAML file path.
// In case it file does not exist or has incorrect YAML:
// - it creates a new default config
//
// Koanf has a file watcher, but its easier to control the changes with a manual reload.
func MustConfig(baseDir string) *Config {
	paths := NewPaths(baseDir)
	filePath := paths.ConfigFile

	fallback := NewDefaultConfig(baseDir)

	k := koanf.New(".")
	provider := file.Provider(filePath)
	if err := k.Load(provider, yaml.Parser()); err != nil {
		return fallback
	}

	cfg := &Config{}
	transformed := cfg.transformConfig(k)
	if err := transformed.Unmarshal("", cfg); err != nil {
		return fallback
	}
	cfg.EnsureConfigValues()
	cfg.App.Paths = paths
	cfg.baseDir = baseDir

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

	if cfg.App == nil {
		return nil, ErrInvalidConfig
	}
	cfg.EnsureConfigValues()

	return cfg, nil
}

func NewDefaultAppConfig(baseDir string) *AppConfig {
	return &AppConfig{
		Port:              2200,
		HomeURL:           "/.ui",
		ServiceURL:        "/.services",
		SettingsURL:       "/.settings",
		ContextURL:        "/.contexts",
		ContextAreaPrefix: "in-",
		SchemaProvider:    DefaultSchemaProvider,
		Paths:             NewPaths(baseDir),
	}
}

// NewDefaultConfig creates a new default config in case the config file is missing, not found or any other error.
func NewDefaultConfig(baseDir string) *Config {
	return &Config{
		App:     NewDefaultAppConfig(baseDir),
		baseDir: baseDir,
	}
}
