package app

import (
    "fmt"
    "github.com/cubahno/xs"
    "github.com/cubahno/xs/api"
    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
    "log"
    "net/http"
    "strings"
    "sync"
    "time"
)

type App struct {
    Router *api.Router
    BluePrints []api.RouteRegister
    mu sync.Mutex
}

func NewApp() *App {
    res := &App{}

    r := chi.NewRouter()
    r.Use(middleware.RequestID)
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)
    r.Use(middleware.Timeout(60 * time.Second))

    config, err := xs.NewConfigFromFile()
    if err != nil {
        log.Printf("Failed to load config file: %s\n", err.Error())
        config = xs.NewDefaultConfig()
    }

    router := &api.Router{
        Mux: r,
        Config: config,
    }
    res.Router = router

    bluePrints := []api.RouteRegister{
        api.CreateHomeRoutes,
        api.LoadServices,
        api.CreateServiceRoutes,
        api.CreateSettingsRoutes,
    }
    res.BluePrints = bluePrints

    for _, bluePrint := range bluePrints {
        err := bluePrint(router)
        if err != nil {
            panic(err)
        }
    }

    return res
}

func (a *App) AddBluePrint(bluePrint api.RouteRegister) error {
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
