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
	"path/filepath"
	"strings"
)

func CreateHomeRoutes(router *Router) error {
	if !router.Config.App.ServeUI {
		return nil
	}
	homeURL := router.Config.App.HomeURL
	url := "/" + strings.Trim(homeURL, "/") + "/"

	homeRedirect := http.RedirectHandler(url, http.StatusMovedPermanently).ServeHTTP
	router.Get(strings.TrimSuffix(url, "/"), homeRedirect)

	router.Get(url, createHomeHandler(router))
	router.Get(url+"export", exportHandler)
	router.Get(url+"import", importHandler)

	fileServer(fmt.Sprintf("/%s/*", strings.Trim(url, "/")), router)
	return nil
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

func exportHandler(w http.ResponseWriter, r *http.Request) {
	// Specify the path to the folder you want to zip
	resourcePath := ResourcePath

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=download.zip")

	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

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

		header.Name, err = filepath.Rel(resourcePath, path)
		if err != nil {
			return err
		}

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

func importHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20) // Set the maximum memory for uploaded files (10 MB)

	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error uploading file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Create a temporary directory to extract files
	tempDir := "temp_extracted"
	err = os.Mkdir(tempDir, 0755)
	if err != nil {
		http.Error(w, "Error creating temporary directory", http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tempDir)

	// Extract the zip file
	zipReader, err := zip.NewReader(file, handler.Size)
	if err != nil {
		http.Error(w, "Error reading zip file", http.StatusInternalServerError)
		return
	}

	for _, zipFile := range zipReader.File {
		filePath := filepath.Join(tempDir, zipFile.Name)

		if zipFile.FileInfo().IsDir() {
			os.MkdirAll(filePath, zipFile.FileInfo().Mode()) // Use zipFile.FileInfo().Mode() instead of zipFile.Mode()
			continue
		}

		writer, err := os.Create(filePath)
		if err != nil {
			http.Error(w, "Error creating file", http.StatusInternalServerError)
			return
		}
		defer writer.Close()

		reader, err := zipFile.Open()
		if err != nil {
			http.Error(w, "Error opening zip file entry", http.StatusInternalServerError)
			return
		}
		defer reader.Close()

		_, err = io.Copy(writer, reader)
		if err != nil {
			http.Error(w, "Error extracting file", http.StatusInternalServerError)
			return
		}
	}

	sourceDir := ResourcePath
	err = filepath.WalkDir(tempDir, func(path string, info os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		fileInfo, err := info.Info() // Fetch the os.FileInfo for the entry
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(tempDir, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(sourceDir, relPath)
		if info.IsDir() {
			os.MkdirAll(targetPath, fileInfo.Mode()) // Use fileInfo.Mode() instead of info.Mode()
		} else {
			source, err := os.Open(path)
			if err != nil {
				return err
			}
			defer source.Close()

			target, err := os.Create(targetPath)
			if err != nil {
				return err
			}
			defer target.Close()

			_, err = io.Copy(target, source)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		http.Error(w, "Error moving extracted files", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Upload and extraction successful")
}
