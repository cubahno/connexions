package api

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/cubahno/connexions/v2/pkg/config"
	"github.com/go-chi/chi/v5"
)

// CreateHomeRoutes creates routes for home.
func CreateHomeRoutes(router *Router) error {
	homeURL := router.Config().HomeURL
	url := "/" + strings.Trim(homeURL, "/") + "/"

	homeRedirect := http.RedirectHandler(url, http.StatusMovedPermanently).ServeHTTP
	router.Get(strings.TrimSuffix(url, "/"), homeRedirect)

	router.Get(url, createUIHandler(router))

	docsServer(fmt.Sprintf("/%s/docs/*", strings.Trim(url, "/")), router)
	fileServer(fmt.Sprintf("/%s/*", strings.Trim(url, "/")), router)

	return nil
}

// BufferedWriter is a writer that captures the response.
// Used to capture the template execution result.
type BufferedWriter struct {
	buf        []byte
	statusCode int
}

// NewBufferedResponseWriter creates a new buffered writer.
func NewBufferedResponseWriter() *BufferedWriter {
	return &BufferedWriter{
		buf: make([]byte, 0, 1024),
	}
}

// Write writes the data to the buffer.
func (bw *BufferedWriter) Write(p []byte) (int, error) {
	bw.buf = append(bw.buf, p...)
	return len(p), nil
}

// Header returns the header.
func (bw *BufferedWriter) Header() http.Header {
	return http.Header{}
}

// WriteHeader writes the status code.
func (bw *BufferedWriter) WriteHeader(statusCode int) {
	bw.statusCode = statusCode
}

// createUIHandler creates a handler function for home.
func createUIHandler(router *Router) http.HandlerFunc {
	uiPath := router.Config().Paths.UI

	return func(w http.ResponseWriter, r *http.Request) {
		tpl, err := template.ParseFiles(fmt.Sprintf("%s/index.html", uiPath))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tmpl := template.Must(tpl, nil)
		cfg := router.Config()

		type TemplateData struct {
			AppConfig *config.AppConfig
			Contents  map[string]template.HTML
			Version   string
		}

		homeContents, err := os.ReadFile(filepath.Join(uiPath, "home.html"))
		if err != nil {
			log.Println("Failed to get home contents", err)
		}

		data := &TemplateData{
			AppConfig: cfg,
			Contents: map[string]template.HTML{
				"Home": template.HTML(homeContents),
			},
			Version: os.Getenv("APP_VERSION"),
		}

		// Create a buffered writer to capture the template execution result.
		buf := NewBufferedResponseWriter()

		err = tmpl.Execute(buf, data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		SendHTML(w, http.StatusOK, buf.buf)
	}
}

// fileServer conveniently sets up a http.FileServer handler to serve
// static files from a http.FileSystem.
func fileServer(url string, router *Router) {
	resDir := router.Config().Paths.Resources
	uiPath := filepath.Join(resDir, "ui")

	router.Get(url, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(http.Dir(uiPath)))
		fs.ServeHTTP(w, r)
	})
}

// docsServer serves the docs assets.
func docsServer(url string, router *Router) {
	router.Get(url, func(w http.ResponseWriter, r *http.Request) {
		fs := http.StripPrefix(
			strings.TrimSuffix(url, "/*"),
			http.FileServer(http.Dir(filepath.Join(router.Config().Paths.Base, "site"))))
		fs.ServeHTTP(w, r)
	})
}
