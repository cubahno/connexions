package main

import (
	"github.com/cubahno/xs"
	"github.com/cubahno/xs/api"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"os"
	"path/filepath"
	"sync"
)

func main() {
	readSpec()
}

func readSpec() {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	r := e.Router()
	err := loadServices(xs.ServicePath, r)
	if err != nil {
		panic(err)
	}

	e.Logger.Fatal(e.Start(":2200"))
}

func loadServices(serviceDirPath string, router *echo.Router) error {
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
				err = api.LoadOpenAPI(filePath, router)
			} else {
				err = api.LoadOverwriteService(filePath, router)
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
