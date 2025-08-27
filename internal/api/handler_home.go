package api

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cubahno/connexions/internal/config"
	"github.com/cubahno/connexions/internal/files"
	"github.com/go-chi/chi/v5"
)

// HomeHandler handles home routes.
type HomeHandler struct {
	*BaseHandler
	router *Router
	mu     sync.Mutex
}

// createHomeRoutes creates routes for home.
// Implements RouteRegister interface.
func createHomeRoutes(router *Router) error {
	if router.Config.App.DisableUI {
		return nil
	}

	handler := &HomeHandler{
		router: router,
	}

	homeURL := router.Config.App.HomeURL
	url := "/" + strings.Trim(homeURL, "/") + "/"

	homeRedirect := http.RedirectHandler(url, http.StatusMovedPermanently).ServeHTTP
	router.Get(strings.TrimSuffix(url, "/"), homeRedirect)

	router.Get(url, createHomeHandlerFunc(router))
	router.Get(url+"export", handler.export)
	router.Post(url+"import", handler.importHandler)
	router.Get(url+"postman", handler.postman)

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

// createHomeHandlerFunc creates a handler function for home.
func createHomeHandlerFunc(router *Router) http.HandlerFunc {
	uiPath := router.Config.App.Paths.UI

	return func(w http.ResponseWriter, r *http.Request) {
		tpl, err := template.ParseFiles(fmt.Sprintf("%s/index.html", uiPath))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tmpl := template.Must(tpl, nil)
		cfg := router.Config.App

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

		NewAPIResponse(w).WithHeader("Content-Type", "text/html; charset=utf-8").Send(buf.buf)
	}
}

// fileServer conveniently sets up a http.FileServer handler to serve
// static files from a http.FileSystem.
func fileServer(url string, router *Router) {
	resDir := router.Config.App.Paths.Resources
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
			http.FileServer(http.Dir(filepath.Join(router.Config.App.Paths.Base, "site"))))
		fs.ServeHTTP(w, r)
	})
}

// export exports the data directory in zip file.
func (h *HomeHandler) export(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	resourcePath := h.router.Config.App.Paths.Data
	asFilename := fmt.Sprintf("connexions-%s.zip", time.Now().Format("2006-01-02"))

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", asFilename))

	zipWriter := zip.NewWriter(w)
	defer func() { _ = zipWriter.Close() }()

	only := []string{
		path.Base(h.router.Config.App.Paths.Services),
		path.Base(h.router.Config.App.Paths.Contexts),
	}

	err := filepath.WalkDir(resourcePath, func(path string, info os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Exclude empty directories
		if info.IsDir() && files.IsEmptyDir(path) {
			return nil
		}

		fileInfo, err := info.Info()
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(fileInfo)
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(resourcePath, path)
		if err != nil {
			return err
		}

		take := false
		for _, include := range only {
			if strings.HasPrefix(rel, include) {
				take = true
				break
			}
		}
		if !take {
			return nil
		}

		header.Name = rel

		if info.IsDir() {
			header.Name += "/"
		}

		zipEntry, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				slog.Error("Failed to open file", "path", path, "error", err)
				return nil
			}
			defer file.Close()

			_, _ = io.Copy(zipEntry, file)
		}

		return nil
	})

	if err != nil {
		http.Error(w, "Failed to create zip file", http.StatusInternalServerError)
	}
}

// importHandler imports the zip file into data directory.
func (h *HomeHandler) importHandler(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	err := r.ParseMultipartForm(512 * 1024 * 1024) // Limit form size to 512 MB
	if err != nil {
		h.JSONResponse(w).WithStatusCode(http.StatusBadRequest).Send(&SimpleResponse{
			Message: err.Error(),
			Success: false,
		})
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		h.JSONResponse(w).WithStatusCode(http.StatusBadRequest).Send(&SimpleResponse{
			Message: err.Error(),
			Success: false,
		})
		return
	}
	defer func() {
		_ = file.Close()
	}()

	zipReader, err := zip.NewReader(file, r.ContentLength)
	if err != nil {
		h.JSONResponse(w).WithStatusCode(http.StatusInternalServerError).Send(&SimpleResponse{
			Message: err.Error(),
		})
		return
	}

	only := []string{
		path.Base(h.router.Config.App.Paths.Plugins),
		path.Base(h.router.Config.App.Paths.Contexts),
		path.Base(h.router.Config.App.Paths.Services),
	}

	err = files.ExtractZip(zipReader, h.router.Config.App.Paths.Data, only)
	if err != nil {
		h.JSONResponse(w).WithStatusCode(http.StatusInternalServerError).Send(&SimpleResponse{
			Message: err.Error(),
		})
		return
	}

	err = loadServices(h.router)
	if err != nil {
		h.JSONResponse(w).WithStatusCode(http.StatusInternalServerError).Send(&SimpleResponse{Message: err.Error()})
		return
	}

	// there's never an error
	_ = loadContexts(h.router)

	if err = loadPlugins(h.router); err != nil {
		h.JSONResponse(w).WithStatusCode(http.StatusInternalServerError).Send(&SimpleResponse{
			Message: err.Error(),
		})
		return
	}

	h.Success("Imported successfully!", w)
}

// postman exports the api resources in Postman collection format.
func (h *HomeHandler) postman(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	asFilename := fmt.Sprintf("connexions-postman-%s.zip", time.Now().Format("2006-01-02"))

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", asFilename))

	services := h.router.GetServices()
	cfg := h.router.Config
	opts := &postmanOptions{
		config:          cfg,
		contexts:        h.router.GetContexts(),
		defaultContexts: h.router.GetDefaultContexts(),
	}
	coll := createPostman(services, opts)
	env := createPostmanEnvironment("cxs[local]", []*PostmanKeyValue{
		{
			Key:   "url",
			Value: fmt.Sprintf("http://localhost:%d", h.router.Config.App.Port),
		},
	})

	callJs, _ := json.MarshalIndent(coll, "", "  ")
	envJs, _ := json.MarshalIndent(env, "", "  ")

	// Create a ZIP writer
	zipWriter := zip.NewWriter(w)
	defer func() { _ = zipWriter.Close() }()

	file1, _ := zipWriter.Create("connexions-collection.json")
	_, _ = file1.Write(callJs)

	file2, _ := zipWriter.Create("connexions-env-loc.json")
	_, _ = file2.Write(envJs)
}
