package internal

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/cubahno/connexions/internal/types"
	"gopkg.in/yaml.v3"
)

type GenerateRequestOptions struct {
	PathPrefix string
	Path       string
	Method     string
	Operation  Operation
}

// GeneratedRequest is a struct that represents a generated GeneratedRequest to be used when building real endpoint GeneratedRequest.
type GeneratedRequest struct {
	Headers       map[string]any  `json:"headers,omitempty"`
	Method        string          `json:"method,omitempty"`
	Path          string          `json:"path,omitempty"`
	Query         string          `json:"query,omitempty"`
	Body          string          `json:"body,omitempty"`
	ContentType   string          `json:"contentType,omitempty"`
	ContentSchema *Schema         `json:"contentSchema,omitempty"`
	Examples      *ContentExample `json:"examples,omitempty"`

	// internal fields. needed for some validation provider.
	Request *http.Request `json:"-"`
}

// ContentExample is a struct that represents a generated cURL example.
type ContentExample struct {
	CURL string `json:"curl,omitempty"`
}

// GeneratedResponse is a struct that represents a generated response to be used when comparing real endpoint response.
type GeneratedResponse struct {
	Headers     http.Header `json:"headers,omitempty"`
	Content     []byte      `json:"content,omitempty"`
	ContentType string      `json:"contentType,omitempty"`
	StatusCode  int         `json:"statusCode,omitempty"`

	// internal fields. needed for some validation provider.
	Operation Operation     `json:"-"`
	Request   *http.Request `json:"-"`
}

// EncodeContent encodes content to the given content type.
// Since it is part of the JSON GeneratedRequest, we need to encode different content types to string before sending it.
func EncodeContent(content any, contentType string) ([]byte, error) {
	if content == nil {
		return nil, nil
	}

	switch contentType {
	case "application/x-www-form-urlencoded", "multipart/form-data", "application/json":
		return json.Marshal(content)

	case "application/xml":
		return xml.Marshal(content)

	case "application/x-yaml":
		return yaml.Marshal(content)

	default:
		switch v := content.(type) {
		case []byte:
			return v, nil
		case string:
			return []byte(v), nil
		}
	}

	return nil, nil
}

func CreateCURLBody(content any, contentType string) (string, error) {
	if content == nil {
		return "", nil
	}

	switch contentType {
	case "application/x-www-form-urlencoded":
		data, ok := content.(map[string]any)
		if !ok {
			return "", ErrUnexpectedFormURLEncodedType
		}

		keys := types.GetSortedMapKeys(data)
		builder := &strings.Builder{}

		for _, key := range keys {
			value := data[key]
			builder.WriteString("--data-urlencode ")
			builder.WriteString(fmt.Sprintf(`'%s=%v'`, url.QueryEscape(key), url.QueryEscape(fmt.Sprintf("%v", value))))
			builder.WriteString(" \\\n")
		}
		return strings.TrimSuffix(builder.String(), " \\\n"), nil

	case "multipart/form-data":
		data, ok := content.(map[string]any)
		if !ok {
			return "", ErrUnexpectedFormDataType
		}

		keys := types.GetSortedMapKeys(data)
		builder := &strings.Builder{}

		for _, key := range keys {
			value := data[key]
			builder.WriteString("--form ")
			builder.WriteString(fmt.Sprintf(`'%s="%v"'`, url.QueryEscape(key), url.QueryEscape(fmt.Sprintf("%v", value))))
			builder.WriteString(" \\\n")
		}
		return strings.TrimSuffix(builder.String(), " \\\n"), nil

	case "application/json":
		enc, err := json.Marshal(content)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("--data-raw '%s'", string(enc)), nil

	case "application/xml":
		enc, err := xml.Marshal(content)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("--data '%s'", string(enc)), nil
	}

	return "", nil
}
