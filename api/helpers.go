package api

import (
	"encoding/json"
	"fmt"
	"github.com/cubahno/xs"
	"net/http"
	"path"
	"path/filepath"
	"strings"
)

type ErrorMessage struct {
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error   string          `json:"error"`
	Details []*ErrorMessage `json:"details"`
}

func GetPayload[T any](req *http.Request) (*T, error) {
	var payload T
	err := json.NewDecoder(req.Body).Decode(&payload)
	if err != nil {
		return nil, err
	}
	return &payload, nil
}

func GetErrorResponse(err error) *ErrorMessage {
	return &ErrorMessage{
		Message: err.Error(),
	}
}

func IsValidHTTPVerb(verb string) bool {
	validVerbs := map[string]bool{
		http.MethodGet:     true,
		http.MethodHead:    true,
		http.MethodPost:    true,
		http.MethodPut:     true,
		http.MethodPatch:   true,
		http.MethodDelete:  true,
		http.MethodConnect: true,
		http.MethodOptions: true,
		http.MethodTrace:   true,
	}

	// Convert the input verb to uppercase for case-insensitive comparison
	verb = strings.ToUpper(verb)

	return validVerbs[verb]
}

func GetPropertiesFromFilePath(filePath string) *FileProperties {
	fileName := path.Base(filePath)
	ext := filepath.Ext(fileName)

	s := strings.TrimPrefix(strings.Replace(filePath, xs.ServicePath, "", 1), "/")
	parts := strings.Split(s, "/")
	serviceName := parts[0]
	method := ""
	resource := ""
	isPossibleOpenAPI := false

	if serviceName == fileName {
		serviceName = strings.TrimSuffix(fileName, ext)
		if ext == ".yaml" || ext == ".yml" || ext == ".json" {
			isPossibleOpenAPI = true
		}
	} else if len(parts) > 1 {
		method = strings.ToUpper(parts[1])
		if !IsValidHTTPVerb(method) {
			method = "GET"
		}
		resource = fmt.Sprintf("/%s/%s", serviceName, strings.Join(parts[2:], "/"))
	}

	return &FileProperties{
		ServiceName:       serviceName,
		IsPossibleOpenAPI: isPossibleOpenAPI,
		Method:            method,
		Resource:          resource,
		FilePath:          filePath,
		FileName:          fileName,
		Extension:         strings.ToLower(ext),
	}
}
