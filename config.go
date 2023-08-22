package xs

import (
	"fmt"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/rawbytes"
	"log"
	"math/rand"
	"strings"
	"time"
)

type Config struct {
	App      *AppConfig                `koanf:"app"`
	Services map[string]*ServiceConfig `koanf:"services"`
}

type ServiceConfig struct {
	Latency  time.Duration          `koanf:"latency"`
	Errors   *ServiceError          `koanf:"errors"`
	Contexts []map[string]string    `koanf:"contexts"`
	Validate *ServiceValidateConfig `koanf:"validate"`
}

type ServiceError struct {
	Chance int         `koanf:"chance"`
	Codes  map[int]int `koanf:"codes"`
}

type ServiceValidateConfig struct {
	Request  bool `koanf:"request"`
	Response bool `koanf:"response"`
}

const (
	RootServiceName = ".root"
	RootOpenAPIName = ".openapi"
)

type AppConfig struct {
	Port              int    `json:"port" koanf:"port"`
	HomeURL           string `json:"homeUrl" koanf:"homeUrl"`
	ServiceURL        string `json:"serviceUrl" koanf:"serviceUrl"`
	SettingsURL       string `json:"settingsUrl" koanf:"settingsUrl"`
	ContextURL        string `json:"contextUrl" koanf:"contextUrl"`
	ContextAreaPrefix string `json:"contextAreaPrefix" koanf:"contextAreaPrefix"`
	ServeUI           bool   `json:"serveUI" koanf:"serveUI"`
	ServeSpec         bool   `json:"serveSpec" koanf:"serveSpec"`
}

func (a *AppConfig) IsValidPrefix(prefix string) bool {
	if prefix == a.HomeURL || prefix == a.ServiceURL || prefix == a.SettingsURL {
		return false
	}
	return true
}

func (c *Config) GetServiceConfig(service string) *ServiceConfig {
	if res, ok := c.Services[service]; ok {
		return res
	}
	return &ServiceConfig{
		Errors: &ServiceError{},
		Validate: &ServiceValidateConfig{
			Request:  true,
			Response: false,
		},
	}
}

func (s *ServiceError) GetError() int {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	randomNumber := rand.Intn(100) + 1
	if randomNumber > s.Chance {
		return 0
	}

	fmt.Printf("Got lucky to throw an error with the %d%% chance\n", s.Chance)

	errorWeights := s.Codes

	// Calculate the total weight
	totalWeight := 0
	for _, weight := range errorWeights {
		totalWeight += weight
	}

	// Generate a random number between 1 and totalWeight
	randomNumber = rand.Intn(totalWeight) + 1

	// Select an error code based on the random number and weights
	for code, weight := range errorWeights {
		randomNumber -= weight
		if randomNumber <= 0 {
			log.Printf("Selected Error Code: %d\n", code)
			return code
		}
	}

	log.Println("Failed to select Error Code")
	return 0
}

// NewConfigFromFile creates a new config from a file path.
// It also creates a watcher for the file and reloads the config on change.
func NewConfigFromFile() (*Config, error) {
	k := koanf.New(".")
	filePath := fmt.Sprintf("%s/config.yml", ResourcePath)
	provider := file.Provider(filePath)
	if err := k.Load(provider, yaml.Parser()); err != nil {
		return nil, err
	}

	cfg := &Config{}
	transformed := TransformConfig(k)
	if err := transformed.Unmarshal("", cfg); err != nil {
		return nil, err
	}

	createConfigWatcher(provider, cfg)
	return cfg, nil
}

// TransformConfig applies transformations to the config.
// Currently, it removes % from the chances.
func TransformConfig(k *koanf.Koanf) *koanf.Koanf {
	transformed := koanf.New(".")
	for key, value := range k.All() {
		if v, isString := value.(string); isString && strings.HasSuffix(v, "%") {
			value = strings.TrimSuffix(v, "%")
		}
		_ = transformed.Set(key, value)
	}
	return transformed
}

func NewConfigFromContent(content []byte) (*Config, error) {
	k := koanf.New(".")
	provider := rawbytes.Provider(content)
	if err := k.Load(provider, yaml.Parser()); err != nil {
		return nil, err
	}

	cfg := &Config{}
	transformed := TransformConfig(k)
	if err := transformed.Unmarshal("", cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func NewDefaultConfig() *Config {
	return &Config{
		App: &AppConfig{
			Port:              2200,
			HomeURL:           "/.ui",
			ServiceURL:        "/.services",
			SettingsURL:       "/.settings",
			ContextURL:        "/.contexts",
			ServeUI:           true,
			ServeSpec:         true,
			ContextAreaPrefix: "-in-",
		},
	}
}

func createConfigWatcher(f *file.File, cfg *Config) {
	f.Watch(func(event interface{}, err error) {
		if err != nil {
			log.Printf("watch error: %v", err)
			return
		}

		// Throw away the old config and load a fresh copy.
		log.Println("config changed. Reloading ...")
		k := koanf.New(".")
		if err := k.Load(f, yaml.Parser()); err != nil {
			log.Printf("error loading config: %v\n", err)
			return
		}

		transformed := TransformConfig(k)
		if err := transformed.Unmarshal("", cfg); err != nil {
			log.Printf("error unmarshalling config: %v\n", err)
			return
		}
		k.Print()

		log.Println("Configuration reloaded!")
		// TODO(igor): replace Sleep
		time.Sleep(1 * time.Second)
	})
}
