package api

import (
	"github.com/cubahno/xs"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/labstack/echo/v4"
	"net/http"
	"regexp"
	"strings"
)

var routePlaceholders = regexp.MustCompile("/:([^/]+)")
var openAPIPlaceholders = regexp.MustCompile("/{([^/]+)}")

type ResourceGeneratePayload struct {
	Resource     string         `json:"resource"`
	Method       string         `json:"method"`
	Replacements map[string]any `json:"replacements"`
}

// LoadOpenAPI loads an OpenAPI specification from a file and adds the routes to the router.
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

	valueMaker := xs.CreateValueMaker()

	for resName, pathItem := range doc.Paths {
		for method, _ := range pathItem.Operations() {
			path := openAPIPlaceholders.ReplaceAllString(prefix+resName, "/:$1")

			// register route
			router.Add(method, path, createResponseHandler(prefix, doc, valueMaker))
		}
	}

	// register resource generator
	router.Add(http.MethodPost, "/services"+prefix, createGenerateOpenAPIResourceHandler(prefix, doc, valueMaker))

	return nil
}

func createGenerateOpenAPIResourceHandler(prefix string, doc *openapi3.T, valueMaker xs.ValueMaker) echo.HandlerFunc {
	return func(c echo.Context) error {
		payload, err := GetPayload[ResourceGeneratePayload](c)
		if err != nil {
			return c.JSON(http.StatusBadRequest, GetErrorResponse(err))
		}

		pathItem := doc.Paths[payload.Resource]
		if pathItem == nil {
			return c.JSON(http.StatusNotFound, GetErrorResponse(ErrResourceNotFound))
		}

		operation := pathItem.GetOperation(strings.ToUpper(payload.Method))
		if operation == nil {
			return c.JSON(http.StatusMethodNotAllowed, GetErrorResponse(ErrResourceMethodNotFound))
		}

		res := map[string]any{}
		res["request"] = xs.NewRequest(prefix, payload.Resource, payload.Method, operation, valueMaker)
		res["response"] = xs.NewResponse(operation, valueMaker)

		return c.JSON(http.StatusOK, res)
	}
}

// createResponseHandler creates a handler function for an OpenAPI route.
func createResponseHandler(prefix string, doc *openapi3.T, valueMaker xs.ValueMaker) echo.HandlerFunc {
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

// openAPIResponseHandler generates the response for a route.
func openAPIResponseHandler(c echo.Context, operation *openapi3.Operation, valueMaker xs.ValueMaker) error {
	response := xs.NewResponse(operation, valueMaker)
	return c.JSON(response.StatusCode, response.Content)
}
