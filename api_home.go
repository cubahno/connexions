package xs

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"html/template"
	"net/http"
	"strings"
)

func CreateHomeRoutes(router *Router) error {
	homeURL := router.Config.App.HomeURL
	url := "/" + strings.Trim(homeURL, "/") + "/"

	homeRedirect := http.RedirectHandler(url, http.StatusMovedPermanently).ServeHTTP
	router.Get(strings.TrimSuffix(url, "/"), homeRedirect)

	router.Get(url, createHomeHandler(router))
	fileServer(fmt.Sprintf("/%s/*", strings.Trim(url, "/")), router)
	return nil
}

func createHomeHandler(router *Router) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//http.ServeFile(w, r, fmt.Sprintf("%s/index.html", xs.UIPath))
		tmpl := template.Must(template.ParseFiles(fmt.Sprintf("%s/index.html", UIPath)))
		config := router.Config.App
		err := tmpl.Execute(w, config)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}
}

// fileServer conveniently sets up a http.FileServer handler to serve
// static files from a http.FileSystem.
func fileServer(url string, r chi.Router) {
	r.Get(url, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(http.Dir(UIPath)))
		fs.ServeHTTP(w, r)
	})
}
