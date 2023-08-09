package main

import (
	"github.com/cubahno/xs/api"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"net/http"
	"time"
)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	bluePrints := []api.RouteRegister{
		api.CreateHomeRoutes,
		api.LoadServices,
		api.CreateServiceRoutes,
		api.CreateSettingsRoutes,
	}

	for _, bluePrint := range bluePrints {
		err := bluePrint(r)
		if err != nil {
			panic(err)
		}
	}

	http.ListenAndServe(":2200", r)
	println("Server started on port 2200")
}
