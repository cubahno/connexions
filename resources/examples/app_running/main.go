package main

import (
	"github.com/cubahno/connexions"
	"net/http"
)

func main() {
	cfg := connexions.NewDefaultConfig("")
	cfg.App.Port = 8888

	app := connexions.NewApp(cfg)

	// add as many blueprints as needed.
	_ = app.AddBluePrint(func(router *connexions.Router) error {
		h := &ApiHandler{connexions.NewBaseHandler()}

		router.HandleFunc("/api/v1/route-1", h.myRoute1)
		router.HandleFunc("/api/v1/route-2", h.myRoute2)
		return nil
	})
	app.Run()
}

type ApiHandler struct {
	*connexions.BaseHandler
}

func (h *ApiHandler) myRoute1(w http.ResponseWriter, r *http.Request) {
	h.JSONResponse(w).WithStatusCode(http.StatusOK).Send(&connexions.SimpleResponse{
		Message: "hello world",
		Success: true,
	})
}

func (h *ApiHandler) myRoute2(w http.ResponseWriter, r *http.Request) {
	h.Response(w).WithHeader("content-type", "text/plain").Send([]byte("hello world"))
}
