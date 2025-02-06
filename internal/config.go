package internal

import (
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/rawbytes"
)

// Config is the main configuration struct.
// App is the app config.
// Services is a map of service name and the corresponding config.
// ServiceName is the first part of the path.
// e.g. /petstore/v1/pets -> petstore
// in case, there's no service name, the name "root" will be used.
type Config struct {
	App      *AppConfig                `koanf:"app" json:"app" yaml:"app"`
	Services map[string]*ServiceConfig `koanf:"services" json:"services" yaml:"services"`
	BaseDir  string                    `koanf:"-"`
	mu       sync.Mutex
}

// ServiceConfig defines the configuration for a particular service.
// Latency is the latency to add to the response.
// Latency not used in the services API, only when endpoint queried directly.
// Errors is the error config.
// Contexts is the list of contexts to use for replacements.
// It is a map of context name defined either in the UI or filename without extension.
// You can refer to the name when building aliases.
// ParseConfig is the config for parsing the OpenAPI spec.
// Validate is the validation config.
// It is used to validate the request and/or response outside the Services API.
// ResponseTransformer is a callback function name which should exist inside callbacks directory and be visible to the plugin.
// Cache is the cache config.
type ServiceConfig struct {
	Upstream            *UpstreamConfig        `koanf:"upstream" json:"upstream" yaml:"upstream"`
	Latency             time.Duration          `koanf:"latency" json:"latency" yaml:"latency"`
	Errors              *ServiceError          `koanf:"errors" json:"errors" yaml:"errors"`
	Contexts            []map[string]string    `koanf:"contexts" json:"contexts" yaml:"contexts"`
	ParseConfig         *ParseConfig           `json:"parseConfig" yaml:"parseConfig" koanf:"parseConfig"`
	Validate            *ServiceValidateConfig `koanf:"validate" json:"validate" yaml:"validate"`
	RequestTransformer  string                 `koanf:"requestTransformer" json:"requestTransformer" yaml:"requestTransformer"`
	ResponseTransformer string                 `koanf:"responseTransformer" json:"responseTransformer" yaml:"responseTransformer"`
	Cache               *ServiceCacheConfig    `koanf:"cache" json:"cache" yaml:"cache"`
}

type UpstreamConfig struct {
	URL     string                `koanf:"url" json:"url" yaml:"url"`
	Headers map[string]string     `koanf:"headers" json:"headers" yaml:"headers"`
	FailOn  *UpstreamFailOnConfig `koanf:"failOn" json:"failOn" yaml:"failOn"`
}

type HttpStatusFailOnConfig []HTTPStatusConfig

type UpstreamFailOnConfig struct {
	TimeOut    time.Duration          `koanf:"timeout" json:"timeout" yaml:"timeout"`
	HTTPStatus HttpStatusFailOnConfig `koanf:"httpStatus" json:"httpStatus" yaml:"httpStatus"`
}

type HTTPStatusConfig struct {
	Exact int    `koanf:"exact" json:"exact" yaml:"exact"`
	Range string `koanf:"range" json:"range" yaml:"range"`
}

// ServiceError defines the error configuration for a service.
// Chance is the chance to return an error.
// In the config, it can be set with %-suffix.
// Codes is a map of error codes and their weights if Chance > 0.
// If no error codes are specified, it returns a 500 error code.
type ServiceError struct {
	Chance int         `koanf:"chance" json:"chance" yaml:"chance"`
	Codes  map[int]int `koanf:"codes" json:"codes" yaml:"codes"`
	mu     sync.Mutex
}

// ServiceValidateConfig defines the validation configuration for a service.
// Request is a flag whether to validate the request.
// Default: true
// Response is a flag whether to validate the response.
// Default: false
type ServiceValidateConfig struct {
	Request  bool `koanf:"request" json:"request" yaml:"request"`
	Response bool `koanf:"response" json:"response" yaml:"response"`
}

// ServiceCacheConfig defines the cache configuration for a service.
// Avoid multiple schema parsing by caching the parsed schema.
// Default: true
type ServiceCacheConfig struct {
	Schema bool `koanf:"schema" json:"schema" yaml:"schema"`
}

// ParseConfig defines the parsing configuration for a service.
// MaxLevels is the maximum level to parse.
// MaxRecursionLevels is the maximum level to parse recursively.
// 0 means no recursion: property will get nil value.
// OnlyRequired is a flag whether to include only required fields.
// If the spec contains deep references, this might significantly speed up parsing.
type ParseConfig struct {
	MaxLevels          int  `koanf:"maxLevels" json:"maxLevels" yaml:"maxLevels"`
	MaxRecursionLevels int  `koanf:"maxRecursionLevels" json:"maxRecursionLevels" yaml:"maxRecursionLevels"`
	OnlyRequired       bool `koanf:"onlyRequired" json:"onlyRequired" yaml:"onlyRequired"`
}

const (
	// RootServiceName is the name and location in the service directory of the service without a name.
	RootServiceName = "root"

	// RootOpenAPIName is the name and location of the OpenAPI service without a name.
	RootOpenAPIName = "openapi"
)

// AppConfig is the app configuration.
// Port is the port number to listen on.
// HomeURL is the URL for the UI home page.
// ServiceURL is the URL for the service and resources endpoints in the UI.
// SettingsURL is the URL for the settings endpoint in the UI.
// ContextURL is the URL for the context endpoint in the UI.
// ContextAreaPrefix sets sub-contexts for replacements in path, header or any other supported place.
//
// for example:
// in-path:
//
//	user_id: "fake:ids.int8"
//
// DisableUI is a flag whether to disable the UI.
// DisableSpec is a flag whether to disable the Swagger UI.
// Paths is the paths to various resource directories.
// CreateFileStructure is a flag whether to create the initial resources file structure:
// contexts, services, etc.
// It will also copy sample files from the samples directory into services.
// Default: true
type AppConfig struct {
	Port                int           `json:"port" yaml:"port" koanf:"port"`
	HomeURL             string        `json:"homeUrl" yaml:"homeURL" koanf:"homeUrl"`
	ServiceURL          string        `json:"serviceUrl" yaml:"serviceURL" koanf:"serviceUrl"`
	SettingsURL         string        `json:"settingsUrl" yaml:"settingsURL" koanf:"settingsUrl"`
	ContextURL          string        `json:"contextUrl" yaml:"contextUrl" koanf:"contextUrl"`
	ContextAreaPrefix   string        `json:"contextAreaPrefix" yaml:"contextAreaPrefix" koanf:"contextAreaPrefix"`
	DisableUI           bool          `json:"disableUI" yaml:"disableUI" koanf:"disableUI"`
	DisableSwaggerUI    bool          `json:"disableSwaggerUI" yaml:"disableSwaggerUI" koanf:"disableSwaggerUI"`
	Paths               *Paths        `json:"-" koanf:"-"`
	CreateFileStructure bool          `koanf:"createFileStructure" json:"createFileStructure" yaml:"createFileStructure"`
	Editor              *EditorConfig `koanf:"editor" json:"editor" yaml:"editor"`
}

type EditorConfig struct {
	Theme    string `koanf:"theme" json:"theme" yaml:"theme"`
	FontSize int    `koanf:"fontSize" json:"fontSize" yaml:"fontSize"`
}

func NewServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		Errors:      NewServiceErrorConfig(),
		ParseConfig: NewParseConfig(),
		Validate:    NewServiceValidateConfig(),
		Cache:       NewServiceCacheConfig(),
	}
}

func NewServiceErrorConfig() *ServiceError {
	return &ServiceError{
		Codes: map[int]int{},
	}
}

func NewServiceValidateConfig() *ServiceValidateConfig {
	return &ServiceValidateConfig{}
}

func NewServiceCacheConfig() *ServiceCacheConfig {
	return &ServiceCacheConfig{
		Schema: true,
	}
}

func NewParseConfig() *ParseConfig {
	return &ParseConfig{
		MaxLevels: 0,
	}
}

func NewPaths(baseDir string) *Paths {
	resDir := filepath.Join(baseDir, "resources")
	dataDir := filepath.Join(resDir, "data")
	svcDir := filepath.Join(dataDir, "services")
	cbDir := filepath.Join(dataDir, "callbacks")

	return &Paths{
		Base:      baseDir,
		Resources: resDir,

		Data:              dataDir,
		Callbacks:         cbDir,
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
	if !ok || res == nil {
		res = NewServiceConfig()
	}

	if res.Errors == nil {
		res.Errors = NewServiceErrorConfig()
	}

	if res.Validate == nil {
		res.Validate = NewServiceValidateConfig()
	}

	if res.Cache == nil {
		res.Cache = NewServiceCacheConfig()
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

	app.Paths = defaultConfig.App.Paths
	c.App = app
}

// transformConfig applies transformations to the config.
// Currently, it removes % from the chances.
func (c *Config) transformConfig(k *koanf.Koanf) *koanf.Koanf {
	transformed := koanf.New(".")
	for key, value := range k.All() {
		envKey := strings.ToUpper(ToSnakeCase(key))
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

func (ss HttpStatusFailOnConfig) Is(status int) bool {
	for _, s := range ss {
		if s.Is(status) {
			return true
		}
	}

	return false
}

func (s *HTTPStatusConfig) Is(status int) bool {
	if s.Exact == status {
		return true
	}

	rangeParts := strings.Split(s.Range, "-")
	if len(rangeParts) != 2 {
		return false
	}

	lower, err1 := strconv.Atoi(rangeParts[0])
	upper, err2 := strconv.Atoi(rangeParts[1])
	if err1 == nil && err2 == nil && status >= lower && status <= upper {
		return true
	}

	return false
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
		log.Printf("error loading config. using fallback: %v\n", err)
		return res
	}

	cfg := res
	transformed := cfg.transformConfig(k)
	if err := transformed.Unmarshal("", cfg); err != nil {
		log.Printf("error loading config. using fallback: %v\n", err)
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

// NewDefaultAppConfig creates a new default app config in case the config file is missing, not found or any other error.
func NewDefaultAppConfig(baseDir string) *AppConfig {
	return &AppConfig{
		Port:              2200,
		HomeURL:           "/.ui",
		ServiceURL:        "/.services",
		SettingsURL:       "/.settings",
		ContextURL:        "/.contexts",
		ContextAreaPrefix: "in-",
		Paths:             NewPaths(baseDir),
		Editor: &EditorConfig{
			Theme:    "chrome",
			FontSize: 16,
		},
	}
}

// NewDefaultConfig creates a new default config in case the config file is missing, not found or any other error.
func NewDefaultConfig(baseDir string) *Config {
	return &Config{
		App:      NewDefaultAppConfig(baseDir),
		Services: make(map[string]*ServiceConfig),
		BaseDir:  baseDir,
	}
}

// Paths is a struct that holds all the paths used by the application.
type Paths struct {
	Base              string
	Resources         string
	Data              string
	Callbacks         string
	Contexts          string
	Docs              string
	Samples           string
	Services          string
	ServicesOpenAPI   string
	ServicesFixedRoot string
	UI                string
	ConfigFile        string
}
