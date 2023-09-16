package connexions

import (
	"archive/zip"
	"fmt"
	"github.com/go-chi/chi/v5"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

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

	docsServer(fmt.Sprintf("/%s/docs/*", strings.Trim(url, "/")), router)
	fileServer(fmt.Sprintf("/%s/*", strings.Trim(url, "/")), router)

	return nil
}

type HomeHandler struct {
	*BaseHandler
	router *Router
	mu     sync.Mutex
}

// bufferedWriter is a writer that captures the response.
// Used to capture the template execution result.
type bufferedWriter struct {
	buf        []byte
	statusCode int
}

func newBufferedResponseWriter() *bufferedWriter {
	return &bufferedWriter{
		buf: make([]byte, 0, 1024),
	}
}

func (bw *bufferedWriter) Write(p []byte) (int, error) {
	bw.buf = append(bw.buf, p...)
	return len(p), nil
}

func (bw *bufferedWriter) Header() http.Header {
	return http.Header{}
}

func (bw *bufferedWriter) WriteHeader(statusCode int) {
	bw.statusCode = statusCode
}

func createHomeHandlerFunc(router *Router) http.HandlerFunc {
	uiPath := router.Config.App.Paths.UI

	return func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.ParseFiles(fmt.Sprintf("%s/index.html", uiPath)))
		config := router.Config.App

		type TemplateData struct {
			AppConfig *AppConfig
			Contents  map[string]template.HTML
		}

		homeContents, err := os.ReadFile(filepath.Join(uiPath, "home.html"))
		if err != nil {
			log.Println("Failed to get home contents", err)
		}

		data := &TemplateData{
			AppConfig: config,
			Contents: map[string]template.HTML{
				"Home": template.HTML(homeContents),
			},
		}

		// Create a buffered writer to capture the template execution result.
		buf := newBufferedResponseWriter()

		err = tmpl.Execute(buf, data)
		if err != nil {
			http.Error(w, ErrInternalServer.Error(), http.StatusInternalServerError)
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

func docsServer(url string, router *Router) {
	router.Get(url, func(w http.ResponseWriter, r *http.Request) {
		fs := http.StripPrefix(
			strings.TrimSuffix(url, "/*"),
			http.FileServer(http.Dir(filepath.Join(router.Config.App.Paths.Base, "site"))))
		fs.ServeHTTP(w, r)
	})
}

func (h *HomeHandler) export(w http.ResponseWriter, r *http.Request) {
	resourcePath := h.router.Config.App.Paths.Resources
	asFilename := fmt.Sprintf("connexions-%s.zip", time.Now().Format("2006-01-02"))

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", asFilename))

	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	only := []string{
		path.Base(h.router.Config.App.Paths.Services),
		path.Base(h.router.Config.App.Paths.Contexts),
	}

	err := filepath.WalkDir(resourcePath, func(path string, info os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Exclude empty directories
		if info.IsDir() && IsEmptyDir(path) {
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
				log.Printf("Failed to open file %s: %s\n", path, err.Error())
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

func (h *HomeHandler) importHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(512 * 1024 * 1024) // Limit form size to 512 MB

	file, _, err := r.FormFile("file")
	if err != nil {
		h.JSONResponse(w).WithStatusCode(http.StatusBadRequest).Send(&SimpleResponse{
			Message: err.Error(),
			Success: false,
		})
		return
	}
	defer file.Close()

	zipReader, err := zip.NewReader(file, r.ContentLength)
	if err != nil {
		h.JSONResponse(w).WithStatusCode(http.StatusInternalServerError).Send(&SimpleResponse{
			Message: err.Error(),
		})
		return
	}

	only := []string{
		path.Base(h.router.Config.App.Paths.Services),
		path.Base(h.router.Config.App.Paths.Contexts),
	}

	err = ExtractZip(zipReader, h.router.Config.App.Paths.Resources, only)
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

	h.success("Imported successfully!", w)
}
