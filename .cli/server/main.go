package main

import (
	"github.com/cubahno/xs"
	"github.com/cubahno/xs/api"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

func main() {
	readSpec()
}

func readSpec() {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	err := loadServices(xs.ServicePath, r)
	if err != nil {
		panic(err)
	}

	http.ListenAndServe(":2200", r)
}

func loadServices(serviceDirPath string, router *chi.Mux) error {
	wg := &sync.WaitGroup{}

	err := filepath.Walk(serviceDirPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if it's a directory
		if info.IsDir() {
			return nil
		}

		inRootPath := serviceDirPath+"/"+info.Name() == filePath

		wg.Add(1)

		go func() {
			defer wg.Done()

			var err error
			if inRootPath {
				err = api.RegisterOpenAPIService(filePath, router)
			} else {
				err = api.RegisterOverwriteService(filePath, router)
			}
			if err != nil {
				println(err.Error())
			}
		}()

		return nil
	})
	wg.Wait()

	return err
}
