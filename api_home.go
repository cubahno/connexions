package xs

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
)

func CreateHomeRoutes(router *Router) error {
	if !router.Config.App.ServeUI {
		return nil
	}

	handler := &HomeHandler{
		router: router,
	}

	homeURL := router.Config.App.HomeURL
	url := "/" + strings.Trim(homeURL, "/") + "/"

	homeRedirect := http.RedirectHandler(url, http.StatusMovedPermanently).ServeHTTP
	router.Get(strings.TrimSuffix(url, "/"), homeRedirect)

	router.Get(url, createHomeHandler(router))
	router.Get(url+"export", exportHandler)
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

func createHomeHandler(router *Router) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

func docsServer(url string, r chi.Router) {
	r.Get(url, func(w http.ResponseWriter, r *http.Request) {
		fs := http.StripPrefix(
			strings.TrimSuffix(url, "/*"),
			http.FileServer(http.Dir(filepath.Join(RootPath, "site"))))
		fs.ServeHTTP(w, r)
	})
}

func exportHandler(w http.ResponseWriter, r *http.Request) {
	// Specify the path to the folder you want to zip
	resourcePath := ResourcePath

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=download.zip")

	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	only := []string{
		path.Base(ServicePath),
		path.Base(ContextPath),
	}

	err := filepath.WalkDir(resourcePath, func(path string, info os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Exclude empty directories
		if info.IsDir() {
			isEmpty, err := IsEmptyDir(path)
			if err != nil {
				return err
			}
			if isEmpty {
				return nil
			}
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
				return err
			}
			defer file.Close()

			_, err = io.Copy(zipEntry, file)
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		log.Println("Error:", err)
		http.Error(w, "Failed to create zip file", http.StatusInternalServerError)
	}
}

func (h *HomeHandler) importHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(512 * 1024 * 1024) // Limit form size to 512 MB

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error uploading file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	zipReader, err := zip.NewReader(file, r.ContentLength)
	if err != nil {
		http.Error(w, "Error reading zip file", http.StatusInternalServerError)
		return
	}

	only := []string{
		path.Base(ServicePath),
		path.Base(ContextPath),
	}

	err = ExtractZip(zipReader, ResourcePath, only)
	if err != nil {
		http.Error(w, "Error extracting and copying files", http.StatusInternalServerError)
		return
	}

	err = LoadServices(h.router)
	if err != nil {
		h.error(500, err.Error(), w)
		return
	}

	h.success("Imported successfully!", w)
}
