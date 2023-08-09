package xs

import (
	"fmt"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/rawbytes"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Services map[string]*ServiceConfig `koanf:"services"`
}

type ServiceConfig struct {
	Latency time.Duration `koanf:"latency"`
	Errors  *ServiceError `koanf:"errors"`
}

type ServiceError struct {
	Chance string         `koanf:"chance"`
	Codes  map[int]string `koanf:"codes"`
}

func (c *Config) GetServiceConfig(service string) *ServiceConfig {
	if res, ok := c.Services[service]; ok {
		return res
	}
	return &ServiceConfig{}
}

func (s *ServiceError) GetError() int {
	chance, err := ParseWeight(s.Chance)
	if err != nil {
		fmt.Printf("Error parsing chance: %v\n", err)
		return 0
	}

	rand.New(rand.NewSource(time.Now().UnixNano()))
	randomNumber := rand.Intn(100) + 1
	if randomNumber > chance {
		return 0
	}

	fmt.Printf("Got lucky to throw an error with the %d%% chance\n", chance)

	errorWeights := s.Codes

	// Calculate the total weight
	totalWeight := 0
	for _, weightStr := range errorWeights {
		weight, err := ParseWeight(weightStr)
		if err != nil {
			fmt.Printf("Error parsing weight: %v\n", err)
			return 0
		}
		totalWeight += weight
	}

	// Generate a random number between 1 and totalWeight
	randomNumber = rand.Intn(totalWeight) + 1

	// Select an error code based on the random number and weights
	for code, weightStr := range errorWeights {
		weight, _ := ParseWeight(weightStr)
		randomNumber -= weight
		if randomNumber <= 0 {
			fmt.Printf("Selected Error Code: %d\n", code)
			return code
		}
	}

	fmt.Print("Failed to select Error Code\n")
	return 0
}

func NewConfigFromFile() (*Config, error) {
	k := koanf.New(".")
	filePath := fmt.Sprintf("%s/config.yml", ResourcePath)
	provider := file.Provider(filePath)
	if err := k.Load(provider, yaml.Parser()); err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := k.Unmarshal("", cfg); err != nil {
		return nil, err
	}

	createConfigWatcher(provider, cfg)
	return cfg, nil
}

func NewConfigFromContent(content []byte) (*Config, error) {
	k := koanf.New(".")
	provider := rawbytes.Provider(content)
	if err := k.Load(provider, yaml.Parser()); err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := k.Unmarshal("", cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func NewDefaultConfig() *Config {
	return &Config{}
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

		if err := k.Unmarshal("", cfg); err != nil {
			log.Printf("error unmarshalling config: %v\n", err)
			return
		}
		k.Print()

		fmt.Println("Configuration reloaded")
		// TODO(igor): replace Sleep
		time.Sleep(1 * time.Second)
	})
}

func ParseWeight(weightStr string) (int, error) {
	weightStr = strings.TrimSuffix(weightStr, "%")
	weight, err := strconv.Atoi(weightStr)
	if err != nil {
		return 0, err
	}
	return weight, nil
}
