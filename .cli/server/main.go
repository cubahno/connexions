package main

import (
    "fmt"
    "github.com/cubahno/xs"
    "github.com/cubahno/xs/api"
    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"
    "os"
    "path/filepath"
    "sync"
    "time"
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

func loadServices(path string, router *echo.Router) error {
    wg := &sync.WaitGroup{}

    err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }

        // Skip if it's a directory
        if info.IsDir() {
            return nil
        }

        inRootPath := path+"/"+info.Name() == filePath
        if inRootPath {
            wg.Add(1)

            go func() {
                defer wg.Done()
                t1 := time.Now()
                fileName := info.Name()
                serviceName := fileName[:len(fileName)-len(filepath.Ext(fileName))]

                err := api.LoadOpenAPI(serviceName, filePath, router)
                if err != nil {
                    println(err.Error())
                }
                t2 := time.Now()
                fmt.Printf("added OpenAPI service %s from %s in %s\n", serviceName, filePath, t2.Sub(t1).String())
            }()
        }

        return nil
    })
    wg.Wait()

    return err
}
