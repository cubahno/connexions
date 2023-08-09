package api

import (
	"fmt"
	"github.com/cubahno/xs"
	"github.com/go-chi/chi/v5"
	"net/http"
	"strings"
)

func CreateHomeRoutes(router *Router) error {
	router.Get("/", homeHandler)

	homeRedirect := http.RedirectHandler("/", http.StatusMovedPermanently).ServeHTTP
	router.Get("/index.html", homeRedirect)
	router.Get("/ui/index.htm", homeRedirect)
	router.Get("/ui", homeRedirect)
	router.Get("/ui/", homeRedirect)

	fileServer("/ui/*", router)
	return nil
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, fmt.Sprintf("%s/index.html", xs.UIPath))
}

// fileServer conveniently sets up a http.FileServer handler to serve
// static files from a http.FileSystem.
func fileServer(path string, r chi.Router) {
	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(http.Dir(xs.UIPath)))
		fs.ServeHTTP(w, r)
	})
}
