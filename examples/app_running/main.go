package main

import (
	"net/http"

	"github.com/cubahno/connexions/internal"
	"github.com/cubahno/connexions/internal/api"
)

func main() {
	cfg := internal.NewDefaultConfig("")
	cfg.App.Port = 8888

	app := api.NewApp(cfg)

	// add as many blueprints as needed.
	_ = app.AddBluePrint(func(router *api.Router) error {
		h := &ApiHandler{api.NewBaseHandler()}

		router.HandleFunc("/api/v1/route-1", h.myRoute1)
		router.HandleFunc("/api/v1/route-2", h.myRoute2)
		return nil
	})
	app.Run()
}

type ApiHandler struct {
	*api.BaseHandler
}

func (h *ApiHandler) myRoute1(w http.ResponseWriter, r *http.Request) {
	h.JSONResponse(w).WithStatusCode(http.StatusOK).Send(&api.SimpleResponse{
		Message: "hello world",
		Success: true,
	})
}

func (h *ApiHandler) myRoute2(w http.ResponseWriter, r *http.Request) {
	h.Response(w).WithHeader("content-type", "text/plain").Send([]byte("hello world"))
}
