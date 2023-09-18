package connexions

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type App struct {
	Router     *Router
	BluePrints []RouteRegister
	Paths      *Paths
	mu         sync.Mutex
}

type Paths struct {
	Base              string
	Resources         string
	Data              string
	Contexts          string
	Docs              string
	Samples           string
	Services          string
	ServicesOpenAPI   string
	ServicesFixedRoot string
	UI                string
	ConfigFile        string
}

func NewApp(config *Config) *App {
	paths := config.App.Paths
	res := &App{
		Paths: paths,
	}
	resourcePath := paths.Resources
	log.Printf("Initing Application. ResourcePath is: %v\n", resourcePath)

	// Seed the random number generator
	rand.New(rand.NewSource(time.Now().UnixNano()))

	if config.App.CreateFileStructure {
		err := MustFileStructure(paths)
		if err != nil {
			panic(err)
		}
		_ = CleanupServiceFileStructure(paths.Services)
	}

	router := NewRouter(config)
	res.Router = router

	bluePrints := []RouteRegister{
		loadServices,
		loadContexts,

		createHomeRoutes,
		createServiceRoutes,
		createContextRoutes,
		createSettingsRoutes,
	}
	res.BluePrints = bluePrints

	for _, bluePrint := range bluePrints {
		err := bluePrint(router)
		if err != nil {
			log.Printf("Failed to load blueprint: %s\n", err.Error())
		}
	}

	return res
}

// MustFileStructure creates the necessary directories and files
func MustFileStructure(paths *Paths) error {
	dirs := []string{paths.Resources, paths.Samples, paths.Data, paths.Services, paths.Contexts}
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.Mkdir(dir, os.ModePerm); err != nil {
				return err
			}

			if dir == paths.Resources {
				log.Println("Copying sample config file")

				srcPath := fmt.Sprintf("%s.dist", paths.ConfigFile)

				// if file system has no config file, create one
				if _, err = os.Stat(paths.ConfigFile); os.IsNotExist(err) {
					// read dist config file
					configContent, err := os.ReadFile(srcPath)
					if err != nil {
						def := &Config{}
						def.EnsureConfigValues()
						configContent, _ = yaml.Marshal(def.App)
					}

					if err := SaveFile(paths.ConfigFile, configContent); err != nil {
						return err
					}
				}
				continue
			}

			if dir == paths.Services {
				log.Println("Copying sample content to service directory")
				if err := CopyDirectory(paths.Samples, paths.Services); err != nil {
					return err
				}
				continue
			}
		}
	}

	log.Println("Done!")
	return nil
}

func (a *App) AddBluePrint(bluePrint RouteRegister) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.BluePrints = append(a.BluePrints, bluePrint)
	err := bluePrint(a.Router)
	if err != nil {
		return err
	}
	return nil
}

func (a *App) Run() {
	config := a.Router.Config
	port := config.App.Port
	homeURL := strings.TrimPrefix(config.App.HomeURL, "/")

	log.Printf("\n\nServer started on port %d. Press Ctrl+C to quit", port)
	log.Printf("Visit http://localhost:%d/%s to view the home page", port, homeURL)

	err := http.ListenAndServe(fmt.Sprintf(":%v", port), a.Router)
	if err != nil {
		panic(err)
	}
}
