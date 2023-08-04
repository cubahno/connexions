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
        for method, _ := range pathItem.Operations() {
            path := openAPIPlaceholders.ReplaceAllString(prefix+resName, "/:$1")
            router.Add(method, path, createHandler(prefix, doc))
        }
    }
    return nil
}

func createHandler(prefix string, doc *openapi3.T) func(c echo.Context) error {
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

        return openAPIHandler(c, operation)
    }
}

func openAPIHandler(c echo.Context, operation *openapi3.Operation) error {
    schemas := operation.Responses
    schemaKey := ""
    j := 0
    for i, _ := range schemas {
        if j == 0 && i != "default" {
            schemaKey = i
            break
        }
        j++
    }
    schema := schemas[schemaKey]

    data := make(map[string]interface{})
    resp := schema.Value.Content["application/json"].Schema.Value
    for key, prop := range resp.Properties {
        if prop.Value != nil {
            data[key] = nestedSchema(prop.Value)
        } else {
            data[key] = ""
        }
    }
    return c.JSON(http.StatusOK, data)

    // return c.JSON(http.StatusOK, map[string]string{"hallo": "welt! " + operation.OperationID})
}
