package xs

import (
    "github.com/getkin/kin-openapi/openapi3"
    "github.com/labstack/echo/v4"
    "net/http"
    "regexp"
    "strings"
)

var routePlaceholders = regexp.MustCompile("/:([^/]+)")
var openAPIPlaceholders = regexp.MustCompile("/{([^/]+)}")

func LoadOpenAPI(serviceName, filePath string, router *echo.Router) error {
    loader := openapi3.NewLoader()
    doc, err := loader.LoadFromFile(filePath)
    if err != nil {
        return err
    }

    prefix := ""
    if serviceName != "" {
        prefix = "/" + serviceName
    }

    for resName, pathItem := range doc.Paths {
        valueMaker := CreateValueMaker()

        for method, _ := range pathItem.Operations() {
            path := openAPIPlaceholders.ReplaceAllString(prefix+resName, "/:$1")
            router.Add(method, path, createResponseHandler(prefix, doc, valueMaker))
        }
    }
    return nil
}

func createResponseHandler(prefix string, doc *openapi3.T, valueMaker ValueMaker) func(c echo.Context) error {
    return func(c echo.Context) error {
        resourceName := strings.Replace(c.Path(), prefix, "", 1)
        resourceName = routePlaceholders.ReplaceAllString(resourceName, "/{$1}")
        paths := doc.Paths[resourceName]

        currentMethod := c.Request().Method
        var operation *openapi3.Operation

        if currentMethod == http.MethodGet {
            operation = paths.Get
        } else if currentMethod == http.MethodPost {
            operation = paths.Post
        } else if currentMethod == http.MethodPut {
            operation = paths.Put
        } else if currentMethod == http.MethodDelete {
            operation = paths.Delete
        } else if currentMethod == http.MethodOptions {
            operation = paths.Options
        } else if currentMethod == http.MethodHead {
            operation = paths.Head
        } else if currentMethod == http.MethodPatch {
            operation = paths.Patch
        } else if currentMethod == http.MethodTrace {
            operation = paths.Trace
        } else {
            return c.NoContent(http.StatusMethodNotAllowed)
        }

        return openAPIResponseHandler(c, operation, valueMaker)
    }
}

func openAPIResponseHandler(c echo.Context, operation *openapi3.Operation, valueMaker ValueMaker) error {
    response := NewResponse(operation, valueMaker)
    return c.JSON(response.StatusCode, response.Content)
}
