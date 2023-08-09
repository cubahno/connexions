package api

import (
	"github.com/go-chi/chi/v5"
	"net/http"
)

func CreateServiceRoutes(router *chi.Mux) error {
	router.Get("/services", serviceListHandler)
	return nil
}

func serviceListHandler(w http.ResponseWriter, r *http.Request) {
	NewJSONResponse(http.StatusOK, []string{"home", "services"}, w)
}
