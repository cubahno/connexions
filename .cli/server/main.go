package main

import (
	"github.com/cubahno/xs"
	"github.com/cubahno/xs/api"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"log"
	"net/http"
	"time"
)

func main() {
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

	bluePrints := []api.RouteRegister{
		api.CreateHomeRoutes,
		api.LoadServices,
		api.CreateServiceRoutes,
		api.CreateSettingsRoutes,
	}

	for _, bluePrint := range bluePrints {
		err := bluePrint(router)
		if err != nil {
			panic(err)
		}
	}

	http.ListenAndServe(":2200", r)
	println("Server started on port 2200")
}
