package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/cubahno/connexions/internal/config"
	"github.com/cubahno/connexions/internal/context"
	"github.com/cubahno/connexions/internal/replacer"
)

type postmanOptions struct {
	config          *config.Config
	contexts        map[string]map[string]any
	defaultContexts []map[string]string
}

func createPostman(services map[string]*ServiceItem, options *postmanOptions) *PostmanCollection {
	// sort by name
	names := make([]string, 0, len(services))
	for name := range services {
		names = append(names, name)
	}
	sort.Strings(names)

	cfg := options.config

	res := &PostmanCollection{
		Info: &PostmanInfo{
			Name:    "Connexions",
			Postman: "postman-id",
			Schema:  "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
		},
	}

	for _, name := range names {
		service := services[name]
		serviceCfg := cfg.GetServiceConfig(name)

		serviceCtxs := serviceCfg.Contexts
		if len(serviceCtxs) == 0 {
			serviceCtxs = options.defaultContexts
		}
		cts := context.CollectContexts(serviceCtxs, options.contexts, nil)
		valueReplacer := replacer.CreateValueReplacer(cfg, replacer.Replacers, cts)

		resourceOptions := &generateResourceOptions{
			config:        serviceCfg,
			valueReplacer: valueReplacer,
			withRequest:   true,
		}

		item := createPostmanCollection(service, resourceOptions)
		if item != nil {
			res.Item = append(res.Item, item)
		}
	}

	res.Item = append(res.Item, createPostmanGeneratorEndpoint())

	return res
}

func createPostmanCollection(service *ServiceItem, opts *generateResourceOptions) *PostmanItem {
	name := service.Name
	if name == "" {
		name = "/"
	}
	item := &PostmanItem{
		Name: name,
		Item: make([]*PostmanItem, 0),
	}

	service.Routes.Sort()

	for _, route := range service.Routes {
		res, err := generateResource(service, route, opts)
		if err != nil {
			slog.Error("Error generating resource", "error", err)
			return nil
		}

		servicePrefix := route.File.Prefix
		if servicePrefix == "/" {
			servicePrefix = ""
		}

		genReq := res.Request
		if genReq == nil {
			slog.Error(fmt.Sprintf("Failed to generate request for %s", route.Path))
			continue
		}

		pmHeader := make([]*PostmanKeyValue, 0)
		for hdrName, hdrValue := range genReq.Headers {
			pmHeader = append(pmHeader, &PostmanKeyValue{
				Key:   hdrName,
				Value: fmt.Sprintf("%v", hdrValue),
			})
		}

		generatedPath := genReq.Path
		generatedPath = strings.TrimPrefix(generatedPath, servicePrefix)
		path, vars := getPostmanPath(servicePrefix, route.Path, generatedPath)

		query := make([]*PostmanKeyValue, 0)
		for _, value := range strings.Split(genReq.Query, "&") {
			parts := strings.Split(value, "=")
			if len(parts) != 2 {
				continue
			}
			query = append(query, &PostmanKeyValue{
				Key:   parts[0],
				Value: parts[1],
			})
		}

		pmRawURL := fmt.Sprintf("{{url}}/%s", strings.Join(path, "/"))
		if genReq.Query != "" {
			pmRawURL += fmt.Sprintf("?%s", genReq.Query)
		}

		pmURL := &PostmanURL{
			Raw:      pmRawURL,
			Host:     []string{"{{url}}"},
			Path:     path,
			Query:    query,
			Variable: vars,
		}

		sub := &PostmanItem{
			Name: route.Path,
			Request: &PostmanRequest{
				Method: res.Request.Method,
				Header: pmHeader,
				Body:   getPostmanBody(genReq.ContentType, genReq.Body),
				URL:    pmURL,
			},
		}
		item.Item = append(item.Item, sub)
	}
	return item
}

func createPostmanGeneratorEndpoint() *PostmanItem {
	body := map[string]any{
		"service":      "service-name",
		"method":       "GET",
		"resource":     "/",
		"replacements": map[string]any{},
	}
	bodyBts, _ := json.Marshal(body)
	pmBody := getPostmanBody("application/json", string(bodyBts))

	return &PostmanItem{
		Name: "Generator",
		Request: &PostmanRequest{
			Method: "POST",
			Header: []*PostmanKeyValue{
				{
					Key:   "Content-Type",
					Value: "application/json",
				},
			},
			Body: pmBody,
			URL: &PostmanURL{
				Raw:  "{{url}}/.services/generate",
				Host: []string{"{{url}}"},
				Path: []string{".services", "generate"},
			},
		},
	}
}

func getPostmanPath(prefix, path, generatedPath string) ([]string, []*PostmanKeyValue) {
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")

	generatedPath = strings.Trim(generatedPath, "/")
	generatedParts := strings.Split(generatedPath, "/")

	pathRes := make([]string, 0)
	if prefix != "/" && prefix != "" {
		// add prefix parts to path
		pathRes = append(pathRes, strings.Split(strings.TrimPrefix(prefix, "/"), "/")...)
	}

	variables := make([]*PostmanKeyValue, 0)

	for i, part := range parts {
		if part == "" {
			continue
		}
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			part = strings.Trim(part, "{}")
			pathRes = append(pathRes, fmt.Sprintf(":%s", part))
			variables = append(variables, &PostmanKeyValue{
				Key:   part,
				Value: generatedParts[i],
			})
		} else {
			pathRes = append(pathRes, part)
		}
	}

	return pathRes, variables
}

func getPostmanBody(contentType string, body string) *PostmanBody {
	if contentType == "application/json" {
		return &PostmanBody{
			Mode: "raw",
			Raw:  prettyPrintPostmanJSON(body),
			Options: &PostmanBodyOptions{
				Raw: &PostmanRawBody{
					Language: "json",
				},
			},
		}
	}

	if contentType == "application/x-www-form-urlencoded" {
		urlencoded := make([]*PostmanKeyValue, 0)
		mapped := make(map[string]string)
		if err := json.Unmarshal([]byte(body), &mapped); err != nil {
			slog.Error("Error unmarshalling form-urlencoded body", "error", err)
			return nil
		}

		for key, value := range mapped {
			urlencoded = append(urlencoded, &PostmanKeyValue{
				Key:   key,
				Value: value,
			})
		}
		return &PostmanBody{
			Mode:       "urlencoded",
			Urlencoded: urlencoded,
		}
	}

	if contentType == "multipart/form-data" {
		formData := make([]*PostmanKeyValue, 0)
		mapped := make(map[string]string)
		if err := json.Unmarshal([]byte(body), &mapped); err != nil {
			slog.Error("Error unmarshalling form-data body", "error", err)
			return nil
		}

		for key, value := range mapped {
			formData = append(formData, &PostmanKeyValue{
				Key:   key,
				Value: value,
			})
		}
		return &PostmanBody{
			Mode:     "formdata",
			FormData: formData,
		}
	}

	return nil
}

func prettyPrintPostmanJSON(jsonStr string) string {
	if jsonStr == "" {
		return ""
	}
	var jsonData map[string]any
	err := json.Unmarshal([]byte(jsonStr), &jsonData)
	if err != nil {
		slog.Error("Error unmarshalling JSON", "error", err)
		return jsonStr
	}

	// Pretty-print JSON with indentation
	prettyJSON, err := json.MarshalIndent(jsonData, "", "    ")
	if err != nil {
		slog.Error("Error marshalling JSON", "error", err)
		return jsonStr
	}

	return string(prettyJSON)
}

func createPostmanEnvironment(name string, values []*PostmanKeyValue) *PostmanEnvironment {
	return &PostmanEnvironment{
		ID:                   fmt.Sprintf("connexions-environment-%s", name),
		Name:                 name,
		Values:               values,
		PostmanVariableScope: "environment",
		PostmanExportedAt:    time.Now().Format("2006-01-02T15:04:05.000Z"),
		PostmanExportedUsing: "connexions",
	}
}

type PostmanCollection struct {
	Info *PostmanInfo   `json:"info"`
	Item []*PostmanItem `json:"item"`
}

type PostmanInfo struct {
	Name    string `json:"name"`
	Postman string `json:"_postman_id"`
	Schema  string `json:"schema"`
}

type PostmanItem struct {
	Name     string             `json:"name"`
	Request  *PostmanRequest    `json:"request,omitempty"`
	Response []*PostmanResponse `json:"response,omitempty"`
	Item     []*PostmanItem     `json:"item,omitempty"`
}

type PostmanRequest struct {
	Method string             `json:"method"`
	Header []*PostmanKeyValue `json:"header"`
	Body   *PostmanBody       `json:"body,omitempty"`
	URL    *PostmanURL        `json:"url,omitempty"`
}

type PostmanBody struct {
	Mode       string              `json:"mode"`
	Raw        string              `json:"raw"`
	Options    *PostmanBodyOptions `json:"options"`
	FormData   []*PostmanKeyValue  `json:"formData,omitempty"`
	Urlencoded []*PostmanKeyValue  `json:"urlencoded,omitempty"`
}

type PostmanBodyOptions struct {
	Raw *PostmanRawBody `json:"raw"`
}

type PostmanRawBody struct {
	Language string `json:"language"`
}

type PostmanURL struct {
	Raw      string             `json:"raw"`
	Host     []string           `json:"host"`
	Path     []string           `json:"path"`
	Query    []*PostmanKeyValue `json:"query,omitempty"`
	Variable []*PostmanKeyValue `json:"variable,omitempty"`
}

type PostmanKeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type PostmanResponse struct {
}

type PostmanEnvironment struct {
	ID                   string             `json:"id"`
	Name                 string             `json:"name"`
	Values               []*PostmanKeyValue `json:"values"`
	PostmanVariableScope string             `json:"_postman_variable_scope"`
	PostmanExportedAt    string             `json:"_postman_exported_at"`
	PostmanExportedUsing string             `json:"_postman_exported_using"`
}
