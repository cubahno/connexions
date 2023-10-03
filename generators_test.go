//go:build !integration

package connexions

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/cubahno/connexions/config"
	"github.com/cubahno/connexions/openapi"
	"github.com/cubahno/connexions/openapi/providers/kin"
	"github.com/cubahno/connexions/replacers"
	"github.com/getkin/kin-openapi/openapi3"
	assert2 "github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

func newOpenAPIParameter(name, in string, schema *openapi.Schema) *openapi.Parameter {
	return &openapi.Parameter{
		Name:   name,
		In:     in,
		Schema: schema,
	}
}

func TestNewRequestFromOperation(t *testing.T) {
	assert := assert2.New(t)

	t.Run("base-case", func(t *testing.T) {
		valueResolver := func(content any, state *replacers.ReplaceState) any {
			schema, _ := content.(*openapi.Schema)
			if state.NamePath[0] == "userId" {
				return "123"
			}
			if schema.Example != nil {
				return schema.Example
			}
			return schema.Default
		}

		operation := &kin.KinOperation{Operation: openapi3.NewOperation()}
		CreateOperationFromYAMLFile(t, filepath.Join("testdata", "operation.yml"), operation)

		req := NewRequestFromOperation("/foo", "/users/{userId}", "POST", operation, valueResolver)

		expectedBodyM := map[string]any{
			"username": "john_doe",
			"email":    "john.doe@example.com",
		}
		expectedBodyB, _ := json.Marshal(expectedBodyM)

		expectedHeaders := map[string]any{"lang": "de"}

		assert.Equal("POST", req.Method)
		assert.Equal("/foo/users/123", req.Path)
		assert.Equal("limit=10", req.Query)
		assert.Equal("application/json", req.ContentType)
		assert.Equal(string(expectedBodyB), req.Body)
		assert.Equal(expectedHeaders, req.Headers)
	})

	t.Run("invalid-resolve-value", func(t *testing.T) {
		valueResolver := func(content any, state *replacers.ReplaceState) any { return func() {} }
		operation := &kin.KinOperation{Operation: openapi3.NewOperation()}
		CreateOperationFromYAMLFile(t, filepath.Join("testdata", "operation-with-invalid-req-body.yml"), operation)

		req := NewRequestFromOperation("/foo", "/users/{userId}", "POST", operation, valueResolver)
		assert.Equal("", req.Body)
	})
}

func TestEncodeContent(t *testing.T) {
	assert := assert2.New(t)

	t.Run("Nil Content", func(t *testing.T) {
		result, err := openapi.EncodeContent(nil, "application/json")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result != nil {
			t.Errorf("Expected empty string, got: %s", result)
		}
	})

	t.Run("String Content", func(t *testing.T) {
		content := "Hello, world!"
		result, err := openapi.EncodeContent(content, "application/json")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if string(result) != fmt.Sprintf(`"%s"`, content) {
			t.Errorf("Expected %s, got: %s", content, result)
		}
	})

	t.Run("JSON Content", func(t *testing.T) {
		content := map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		}
		expectedResult := `{"key1":"value1","key2":42}`
		result, err := openapi.EncodeContent(content, "application/json")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if string(result) != expectedResult {
			t.Errorf("Expected '%s', got: %s", expectedResult, result)
		}
	})

	t.Run("XML Content", func(t *testing.T) {
		type Data struct {
			Name     string `json:"name" xml:"name"`
			Age      int    `json:"age" xml:"age"`
			Settings string `json:"settings" xml:"settings"`
		}
		structContent := Data{
			Name:     "John Doe",
			Age:      30,
			Settings: "some settings",
		}

		result, err := openapi.EncodeContent(structContent, "application/xml")
		assert.NoError(err)

		expectedXML, err := xml.Marshal(structContent)
		assert.NoError(err)

		assert.Equal(string(expectedXML), string(result))
	})

	t.Run("YAML Content", func(t *testing.T) {
		type Data struct {
			Name     string `json:"name" yaml:"name"`
			Age      int    `json:"age" yaml:"age"`
			Settings string `json:"settings" yaml:"settings"`
		}
		structContent := Data{
			Name:     "John Doe",
			Age:      30,
			Settings: "some settings",
		}

		result, err := openapi.EncodeContent(structContent, "application/x-yaml")
		assert.NoError(err)

		expectedYAML, err := yaml.Marshal(structContent)
		assert.NoError(err)

		assert.Equal(string(expectedYAML), string(result))
	})

	t.Run("Byte Content", func(t *testing.T) {
		content := []byte("hallo, welt!")
		result, err := openapi.EncodeContent(content, "x-unknown")
		assert.NoError(err)
		assert.Equal(content, result)
	})

	t.Run("string-content", func(t *testing.T) {
		content := "hallo, welt!"
		result, err := openapi.EncodeContent(content, "x-unknown")
		assert.NoError(err)
		assert.Equal(content, string(result))
	})

	t.Run("Unknown Content Type", func(t *testing.T) {
		content := 123
		result, err := openapi.EncodeContent(content, "x-unknown")
		assert.NoError(err)
		assert.Nil(result)
	})
}

func TestCreateCURLBody(t *testing.T) {
	assert := assert2.New(t)

	t.Run("nil-content", func(t *testing.T) {
		res, err := openapi.CreateCURLBody(nil, "application/json")
		assert.NoError(err)
		assert.Equal("", res)
	})

	t.Run("FormURLEncoded", func(t *testing.T) {
		t.Parallel()

		content := map[string]interface{}{
			"name":  "John",
			"age":   30,
			"email": "john@example.com",
		}

		result, err := openapi.CreateCURLBody(content, "application/x-www-form-urlencoded")
		assert.NoError(err)

		expected := `--data-urlencode 'age=30' \
--data-urlencode 'email=john%40example.com' \
--data-urlencode 'name=John'
`
		expected = strings.TrimSuffix(expected, "\n")
		assert.Equal(expected, result)
	})

	t.Run("FormURLEncoded-incorrect-content-type", func(t *testing.T) {
		t.Parallel()

		// should be map[string]any
		content := map[string]string{
			"name":  "John",
			"email": "john@example.com",
		}

		result, err := openapi.CreateCURLBody(content, "application/x-www-form-urlencoded")
		assert.Equal("", result)
		assert.Equal(ErrUnexpectedFormURLEncodedType, err)
	})

	t.Run("MultipartFormData", func(t *testing.T) {
		t.Parallel()

		content := map[string]interface{}{
			"name":  "Jane",
			"age":   25,
			"email": "jane@example.com",
		}

		result, err := openapi.CreateCURLBody(content, "multipart/form-data")
		assert.NoError(err)

		expected := `--form 'age="25"' \
--form 'email="jane%40example.com"' \
--form 'name="Jane"'
`
		expected = strings.TrimSuffix(expected, "\n")
		assert.Equal(expected, result)
	})

	t.Run("MultipartFormData-incorrect-content-type", func(t *testing.T) {
		t.Parallel()

		// should be map[string]any
		content := map[string]string{
			"name":  "Jane",
			"email": "jane@example.com",
		}

		result, err := openapi.CreateCURLBody(content, "multipart/form-data")
		assert.Equal("", result)
		assert.Equal(ErrUnexpectedFormDataType, err)
	})

	t.Run("JSON", func(t *testing.T) {
		t.Parallel()

		content := map[string]interface{}{
			"name":  "Alice",
			"age":   28,
			"email": "alice@example.com",
		}

		result, err := openapi.CreateCURLBody(content, "application/json")
		assert.NoError(err)

		enc, _ := json.Marshal(content)
		expected := fmt.Sprintf("--data-raw '%s'", string(enc))
		assert.Equal(expected, result)
	})

	t.Run("XML", func(t *testing.T) {
		t.Parallel()

		type Person struct {
			Name  string `xml:"name"`
			Age   int    `xml:"age"`
			Email string `xml:"email"`
		}

		content := Person{
			Name:  "Eve",
			Age:   22,
			Email: "eve@example.com",
		}

		result, err := openapi.CreateCURLBody(content, "application/xml")
		assert.NoError(err)

		enc, _ := xml.Marshal(content)
		expected := fmt.Sprintf("--data '%s'", string(enc))
		assert.Equal(expected, result)
	})

	t.Run("XML-invalid", func(t *testing.T) {
		t.Parallel()

		result, err := openapi.CreateCURLBody(func() {}, "application/xml")
		assert.Equal("", result)
		assert.Error(err)
	})

	t.Run("unknown-content-type", func(t *testing.T) {
		t.Parallel()

		result, err := openapi.CreateCURLBody(123, "application/unknown")
		assert.Equal("", result)
		assert.NoError(err)
	})
}

func TestNewRequestFromFixedResource(t *testing.T) {
	assert := assert2.New(t)
	valueReplacer := func(content any, state *replacers.ReplaceState) any {
		return "resolved-value"
	}
	res := NewRequestFromFixedResource("/foo/bar", http.MethodPatch, "application/json", valueReplacer)
	expected := &openapi.GeneratedRequest{
		Method:      http.MethodPatch,
		Path:        "/foo/bar",
		ContentType: "application/json",
		Examples:    nil,
	}
	assert.Equal(expected, res)
}

func TestNewResponseFromOperation(t *testing.T) {
	assert := assert2.New(t)

	t.Run("base-case", func(t *testing.T) {
		valueResolver := func(content any, state *replacers.ReplaceState) any {
			schema, _ := content.(*openapi.Schema)
			if state.NamePath[0] == "userId" {
				return 123
			}
			if schema.Example != nil {
				return schema.Example
			}
			return schema.Default
		}

		operation := &kin.KinOperation{Operation: openapi3.NewOperation()}
		CreateOperationFromYAMLFile(t, filepath.Join("testdata", "operation-base.yml"), operation)
		r, _ := http.NewRequest(http.MethodGet, "/api/resources/1", nil)
		res := NewResponseFromOperation(r, operation, valueResolver)

		expectedHeaders := http.Header{
			"Location":     []string{"https://example.com/users/123"},
			"Content-Type": []string{"application/json"},
		}
		expectedContentM := map[string]any{
			"id":    float64(123),
			"email": "jane.doe@example.com",
		}
		expectedContent, _ := json.Marshal(expectedContentM)

		assert.Equal("application/json", res.ContentType)
		assert.Equal(200, res.StatusCode)
		assert.Equal(expectedHeaders, res.Headers)
		assert.Equal(expectedContent, res.Content)
	})

	t.Run("no-content-type", func(t *testing.T) {
		valueResolver := func(content any, state *replacers.ReplaceState) any {
			schema, _ := content.(*openapi.Schema)
			if state.NamePath[0] == "userId" {
				return 123
			}
			if schema.Example != nil {
				return schema.Example
			}
			return schema.Default
		}

		operation := &kin.KinOperation{Operation: openapi3.NewOperation()}
		CreateOperationFromYAMLFile(t, filepath.Join("testdata", "operation-without-content-type.yml"), operation)

		r, _ := http.NewRequest(http.MethodGet, "/api/resources/1", nil)
		res := NewResponseFromOperation(r, operation, valueResolver)

		expectedHeaders := http.Header{
			"Content-Type": []string{"application/json"},
			"Location":     []string{"https://example.com/users/123"},
		}

		assert.Equal(200, res.StatusCode)
		assert.Equal(expectedHeaders, res.Headers)

		assert.Equal("application/json", res.ContentType)
		assert.Nil(res.Content)
	})

	t.Run("invalid-resolved-value", func(t *testing.T) {
		valueResolver := func(content any, state *replacers.ReplaceState) any {
			if state.NamePath[0] == "userId" {
				return 123
			}
			return func() {}
		}

		operation := &kin.KinOperation{Operation: openapi3.NewOperation()}
		CreateOperationFromYAMLFile(t, filepath.Join("testdata", "operation-base.yml"), operation)

		r, _ := http.NewRequest(http.MethodGet, "/api/resources/1", nil)
		res := NewResponseFromOperation(r, operation, valueResolver)
		assert.Nil(res.Content)
	})
}

func TestNewResponseFromFixedResponse(t *testing.T) {
	assert := assert2.New(t)

	t.Run("happy-path", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "users.json")
		fileContent := []byte(`[{"name":"Jane"},{"name":"John"}]`)

		err := SaveFile(filePath, fileContent)
		assert.Nil(err)

		res := NewResponseFromFixedResource(filePath, "application/json", nil)
		expected := &openapi.GeneratedResponse{
			Headers:     http.Header{"Content-Type": []string{"application/json"}},
			Content:     fileContent,
			ContentType: "application/json",
			StatusCode:  http.StatusOK,
		}
		assert.Equal(expected, res)
	})

	t.Run("bad-json", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "users.json")
		fileContent := []byte(`[{"name":"Jane"}`)

		err := SaveFile(filePath, fileContent)
		assert.Nil(err)

		res := NewResponseFromFixedResource(filePath, "application/json", nil)
		expected := &openapi.GeneratedResponse{
			Headers:     http.Header{"Content-Type": []string{"application/json"}},
			Content:     nil,
			ContentType: "application/json",
			StatusCode:  http.StatusOK,
		}
		assert.Equal(expected, res)
	})

	t.Run("bad-xml", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "users.xml")
		fileContent := []byte(`<name>`)

		err := SaveFile(filePath, fileContent)
		assert.Nil(err)

		res := NewResponseFromFixedResource(filePath, "application/xml", nil)
		expected := &openapi.GeneratedResponse{
			Headers:     http.Header{"Content-Type": []string{"application/xml"}},
			Content:     []byte("<name>"),
			ContentType: "application/xml",
			StatusCode:  http.StatusOK,
		}
		assert.Equal(expected, res)
	})

	t.Run("file-not-found", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "users.xml")

		res := NewResponseFromFixedResource(filePath, "application/xml", nil)
		expected := &openapi.GeneratedResponse{
			Headers:     http.Header{"Content-Type": []string{"application/xml"}},
			Content:     nil,
			ContentType: "application/xml",
			StatusCode:  http.StatusOK,
		}
		assert.Equal(expected, res)
	})

	t.Run("empty-filepath", func(t *testing.T) {
		res := NewResponseFromFixedResource("", "application/xml", nil)
		expected := &openapi.GeneratedResponse{
			Headers:     http.Header{"Content-Type": []string{"application/xml"}},
			Content:     nil,
			ContentType: "application/xml",
			StatusCode:  http.StatusOK,
		}
		assert.Equal(expected, res)
	})
}

func TestGenerateURLFromSchemaParameters(t *testing.T) {
	assert := assert2.New(t)

	t.Run("params correctly replaced in path", func(t *testing.T) {
		path := "/users/{id}/{file-id}"
		valueResolver := func(content any, state *replacers.ReplaceState) any {
			if state.NamePath[0] == "id" {
				return 123
			}
			if state.NamePath[0] == "file-id" {
				return "foo"
			}
			return "something-else"
		}

		params := openapi.Parameters{
			newOpenAPIParameter("id", "path", CreateSchemaFromString(t, `{"type": "integer"}`)),
			newOpenAPIParameter("file-id", "path", CreateSchemaFromString(t, `{"type": "string"}`)),
			newOpenAPIParameter("file-id", "query", CreateSchemaFromString(t, `{"type": "integer"}`)),
		}
		res := GenerateURLFromSchemaParameters(path, valueResolver, params)
		assert.Equal("/users/123/foo", res)
	})

	t.Run("replaced-with-empty-dont-happen", func(t *testing.T) {
		path := "/users/{id}/{file-id}"
		valueResolver := func(content any, state *replacers.ReplaceState) any { return "" }

		params := openapi.Parameters{
			newOpenAPIParameter("id", "path", CreateSchemaFromString(t, `{"type": "integer"}`)),
			newOpenAPIParameter("file-id", "path", CreateSchemaFromString(t, `{"type": "string"}`)),
		}
		res := GenerateURLFromSchemaParameters(path, valueResolver, params)
		assert.Equal("/users/{id}/{file-id}", res)
	})
}

func TestGenerateURLFromFixedResourcePath(t *testing.T) {
	assert := assert2.New(t)

	t.Run("without-value-replacer", func(t *testing.T) {
		res := generateURLFromFixedResourcePath("/users/{id}/{file-id}", nil)
		assert.Equal("/users/{id}/{file-id}", res)
	})

	t.Run("happy-path", func(t *testing.T) {
		valueReplacer := func(schema any, state *replacers.ReplaceState) any {
			if state.NamePath[0] == "id" {
				return 123
			}
			if state.NamePath[0] == "file-id" {
				return "foo"
			}
			return ""
		}

		res := generateURLFromFixedResourcePath("/users/{id}/{file-id}/{action}", valueReplacer)
		assert.Equal("/users/123/foo/{action}", res)
	})
}

func TestGenerateQuery(t *testing.T) {
	assert := assert2.New(t)

	t.Run("params correctly replaced in query", func(t *testing.T) {
		valueResolver := func(schema any, state *replacers.ReplaceState) any {
			if state.NamePath[0] == "id" {
				return 123
			}
			if state.NamePath[0] == "file-id" {
				return "foo"
			}
			return "something-else"
		}
		params := openapi.Parameters{
			newOpenAPIParameter("id", "query", CreateSchemaFromString(t, `{"type": "integer"}`)),
			newOpenAPIParameter("file-id", "query", CreateSchemaFromString(t, `{"type": "foo"}`)),
		}
		res := GenerateQuery(valueResolver, params)

		// TODO(cubahno): fix order of query params
		assert.Contains([]string{"id=123&file-id=foo", "file-id=foo&id=123"}, res)
	})

	t.Run("arrays in url", func(t *testing.T) {
		valueResolver := func(content any, state *replacers.ReplaceState) any {
			return "foo bar"
		}
		params := openapi.Parameters{
			newOpenAPIParameter(
				"tags",
				"query",
				CreateSchemaFromString(t, `{"type": "array", "items": {"type": "string"}}`),
			),
		}
		res := GenerateQuery(valueResolver, params)

		expected := "tags[]=foo+bar"
		assert.Equal(expected, res)
	})

	t.Run("no-resolved-values", func(t *testing.T) {
		valueResolver := func(schema any, state *replacers.ReplaceState) any {
			return nil
		}
		params := openapi.Parameters{
			newOpenAPIParameter(
				"id",
				"query",
				CreateSchemaFromString(t, `{"type": "integer"}`),
			),
		}
		res := GenerateQuery(valueResolver, params)

		expected := "id="
		assert.Equal(expected, res)
	})
}

func TestGenerateContentFromSchema(t *testing.T) {
	assert := assert2.New(t)
	cfg := config.NewDefaultConfig("")

	t.Run("base-case", func(t *testing.T) {
		valueResolver := func(content any, state *replacers.ReplaceState) any {
			switch state.NamePath[len(state.NamePath)-1] {
			case "id":
				return 21
			case "score":
				return 11.5
			case "limit":
				return 100
			case "tag1":
				return "#dice"
			case "tag2":
				return "#nice"
			case "offset":
				return -1
			case "query":
				return "games"
			case "first":
				return 10
			case "second":
				return 20
			case "last":
				return 30
			}
			return nil
		}

		target := openapi3.NewSchema()
		CreateSchemaFromYAMLFile(t, filepath.Join("testdata", "schema-base.yml"), target)
		schema := kin.NewSchemaFromKin(target, nil)

		res := GenerateContentFromSchema(schema, valueResolver, nil)

		expected := map[string]any{
			"user": map[string]any{"id": 21, "score": 11.5},
			"pages": []any{
				map[string]any{
					"limit":  100,
					"tag1":   "#dice",
					"tag2":   "#nice",
					"offset": -1,
					"first":  10,
					"second": 20,
				},
			},
		}
		assert.Equal(expected, res)
	})

	t.Run("with-empty-not-nullable-array", func(t *testing.T) {
		valueResolver := func(schema any, state *replacers.ReplaceState) any {
			return replacers.NULL
		}
		schema := CreateSchemaFromString(t, `
        {
            "type": "array",
            "items": {
                "type": "string"
            }
        }`)
		res := GenerateContentFromSchema(schema, valueResolver, nil)

		expected := make([]any, 0)
		assert.Equal(expected, res)
	})

	t.Run("with-empty-but-nullable-array", func(t *testing.T) {
		valueResolver := func(schema any, state *replacers.ReplaceState) any {
			return replacers.NULL
		}
		schema := CreateSchemaFromString(t, `
        {
            "type": "array",
			"nullable": true,
            "items": {
                "type": "string"
            }
        }`)
		res := GenerateContentFromSchema(schema, valueResolver, nil)
		assert.Nil(res)
	})

	t.Run("fast-track-resolve-null-string", func(t *testing.T) {
		valueResolver := func(schema any, state *replacers.ReplaceState) any {
			return replacers.NULL
		}
		schema := CreateSchemaFromString(t, `
        {
            "type": "string"
        }`)
		res := GenerateContentFromSchema(schema, valueResolver, replacers.NewReplaceStateWithName("name"))
		assert.Nil(res)
	})

	t.Run("with-nested-all-of", func(t *testing.T) {
		valueResolver := func(schema any, state *replacers.ReplaceState) any {
			switch state.NamePath[len(state.NamePath)-1] {
			case "name":
				return "Jane Doe"
			case "age":
				return 30
			case "tag":
				return "#doe"
			case "league":
				return "premier"
			case "rating":
				return 345.6
			}
			return nil
		}

		target := openapi3.NewSchema()
		CreateSchemaFromYAMLFile(t, filepath.Join("testdata", "schema-with-nested-all-of.yml"), target)
		schema := kin.NewSchemaFromKin(target, nil)

		expected := map[string]any{"name": "Jane Doe", "age": 30, "tag": "#doe", "league": "premier", "rating": 345.6}

		res := GenerateContentFromSchema(schema, valueResolver, nil)
		assert.Equal(expected, res)
	})

	t.Run("fast-track-not-used-with-object", func(t *testing.T) {
		dice := map[string]string{"nice": "very nice", "rice": "good rice"}

		valueResolver := func(schema any, state *replacers.ReplaceState) any {
			switch state.NamePath[len(state.NamePath)-1] {
			case "nice":
				return "not so nice"
			case "rice":
				return "not a rice"
			case "dice":
				return dice
			}
			return nil
		}
		schema := CreateSchemaFromString(t, `
        {
            "type":"object",
            "properties": {
                "dice": {
                    "type": "object",
                    "properties": {
                        "nice": {
                            "type": "string"
                        },
                        "rice": {
                            "type": "string"
                        }
                    }
                }
            }
        }`)
		res := GenerateContentFromSchema(schema, valueResolver, nil)

		expected := map[string]any{"dice": map[string]any{
			"nice": "not so nice",
			"rice": "not a rice",
		}}
		assert.Equal(expected, res)
	})

	t.Run("with-circular-array-references", func(t *testing.T) {
		valueResolver := func(schema any, state *replacers.ReplaceState) any {
			switch state.NamePath[len(state.NamePath)-1] {
			case "id":
				return 123
			case "name":
				return "noda-123"
			}
			return nil
		}

		filePath := filepath.Join("testdata", "document-with-circular-array.yml")
		doc, err := kin.NewDocumentFromFile(filePath)
		assert.Nil(err)

		resp := doc.FindOperation(&openapi.OperationDescription{Resource: "/nodes/{id}", Method: http.MethodGet}).GetResponse()
		schema := resp.Content
		res := GenerateContentFromSchema(schema, valueResolver, nil)

		expected := map[string]any{
			"id":   123,
			"name": "noda-123",
			"children": []any{
				map[string]any{
					"id":   123,
					"name": "noda-123",
				},
			},
		}
		assert.Equal(expected, res)
	})

	t.Run("with-circular-object-references", func(t *testing.T) {
		valueResolver := func(schema any, state *replacers.ReplaceState) any {
			switch state.NamePath[len(state.NamePath)-1] {
			case "id":
				return 123
			case "name":
				return "noda-123"
			}
			return nil
		}
		filePath := filepath.Join("testdata", "document-circular-with-references.yml")
		doc, err := kin.NewDocumentFromFile(filePath)
		assert.Nil(err)

		resp := doc.FindOperation(&openapi.OperationDescription{Resource: "/nodes/{id}", Method: http.MethodGet}).GetResponse()
		schema := resp.Content
		res := GenerateContentFromSchema(schema, valueResolver, nil)

		expected := map[string]any{
			"id":   123,
			"name": "noda-123",
			"parent": map[string]any{
				"id":   123,
				"name": "noda-123",
			},
		}
		assert.Equal(expected, res)
	})

	t.Run("with-circular-object-references-inlined", func(t *testing.T) {
		valueResolver := func(schema any, state *replacers.ReplaceState) any {
			switch state.NamePath[len(state.NamePath)-1] {
			case "id":
				return 123
			case "name":
				return "noda-123"
			}
			return nil
		}
		filePath := filepath.Join("testdata", "document-circular-with-inline.yml")
		doc, err := kin.NewDocumentFromFile(filePath)
		assert.Nil(err)

		resp := doc.FindOperation(&openapi.OperationDescription{Resource: "/nodes/{id}", Method: http.MethodGet}).GetResponse()
		schema := resp.Content
		res := GenerateContentFromSchema(schema, valueResolver, nil)

		expected := map[string]any{
			"id":   123,
			"name": "noda-123",
			"parent": map[string]any{
				"id":     123,
				"name":   "noda-123",
				"parent": map[string]any{},
			},
		}
		assert.Equal(expected, res)
	})

	t.Run("with-circular-level-1", func(t *testing.T) {
		valueReplacer := replacers.CreateValueReplacer(cfg, replacers.Replacers, nil)
		filePath := filepath.Join("testdata", "document-circular-ucr.yml")
		doc, err := kin.NewDocumentFromFile(filePath)
		assert.Nil(err)

		operation := doc.FindOperation(&openapi.OperationDescription{Resource: "/api/org-api/v1/organization/{acctStructureCode}", Method: http.MethodGet})
		operation.WithParseConfig(&config.ParseConfig{MaxRecursionLevels: 1})
		resp := operation.GetResponse()
		schema := resp.Content
		res := GenerateContentFromSchema(schema, valueReplacer, nil)

		orgs := []string{"Division", "Department", "Organization"}
		v := res.(map[string]any)

		assert.NotNil(res)
		assert.Contains([]bool{true, false}, v["success"])

		r := v["response"].(map[string]any)
		parent := r["parent"].(map[string]any)
		assert.Contains(orgs, parent["type"])
		assert.NotNil(parent["children"])
		assert.Equal(1, len(parent["children"].([]any)))
		assert.Nil(parent["parent"])

		typ := r["type"]
		assert.Contains(orgs, typ)

		children := r["children"].([]any)
		assert.Equal(1, len(children))
		kid := children[0].(map[string]any)
		assert.Contains(orgs, kid["type"])
		assert.NotNil(kid["children"])
		assert.Equal(1, len(kid["children"].([]any)))
		assert.Nil(kid["parent"])
	})
}

func TestGenerateContentFromSchema_ReadWrite(t *testing.T) {
	assert := assert2.New(t)

	t.Run("read-only-complete-object-when-write-only-requested", func(t *testing.T) {
		valueResolver := func(schema any, state *replacers.ReplaceState) any {
			name := state.NamePath[len(state.NamePath)-1]
			return name + "-value"
		}
		schema := CreateSchemaFromString(t, `
        {
            "type":"object",
            "properties": {
                "product": {
                    "type": "object",
                    "properties": {
                        "nice": {
                            "type": "string"
                        },
                        "rice": {
                            "type": "string"
                        },
						"price": {
							"type": "string"
						}
                    }
                }
            },
			"readOnly": true
        }`)
		state := replacers.NewReplaceState(replacers.WithWriteOnly())
		res := GenerateContentFromSchema(schema, valueResolver, state)

		assert.Nil(res)
	})

	t.Run("read-only-inner-object", func(t *testing.T) {
		valueResolver := func(schema any, state *replacers.ReplaceState) any {
			name := state.NamePath[len(state.NamePath)-1]
			return name + "-value"
		}
		schema := CreateSchemaFromString(t, `
        {
            "type":"object",
            "properties": {
                "product": {
                    "type": "object",
                    "properties": {
                        "nice": {
                            "type": "string"
                        },
                        "rice": {
                            "type": "string"
                        },
						"price": {
							"type": "string"
						}
                    },
					"readOnly": true
                }
            }
        }`)
		state := replacers.NewReplaceState(replacers.WithWriteOnly())
		res := GenerateContentFromSchema(schema, valueResolver, state)

		expected := map[string]any{}
		assert.Equal(expected, res)
	})

	t.Run("read-only-properties", func(t *testing.T) {
		valueResolver := func(schema any, state *replacers.ReplaceState) any {
			name := state.NamePath[len(state.NamePath)-1]
			return name + "-value"
		}
		schema := CreateSchemaFromString(t, `
        {
            "type":"object",
            "properties": {
                "product": {
                    "type": "object",
                    "properties": {
                        "nice": {
                            "type": "string",
							"readOnly": true
                        },
                        "rice": {
                            "type": "string",
							"writeOnly": true
                        },
						"price": {
							"type": "string"
						}
                    }
                }
            }
        }`)
		state := replacers.NewReplaceState(replacers.WithReadOnly())
		res := GenerateContentFromSchema(schema, valueResolver, state)

		// only ro included
		expected := map[string]any{
			"product": map[string]any{
				"nice":  "nice-value",
				"price": "price-value",
			},
		}
		assert.Equal(expected, res)
	})
}

func TestGenerateContentObject(t *testing.T) {
	assert := assert2.New(t)

	t.Run("GenerateContentObject", func(t *testing.T) {
		target := openapi3.NewSchema()
		CreateSchemaFromYAMLFile(t, filepath.Join("testdata", "schema-with-name-obj-and-age.yml"), target)
		schema := kin.NewSchemaFromKin(target, nil)

		valueResolver := func(schema any, state *replacers.ReplaceState) any {
			namePath := state.NamePath
			for _, name := range namePath {
				if name == "first" {
					return "Jane"
				} else if name == "last" {
					return "Doe"
				} else if name == "age" {
					return 21
				}
			}
			return nil
		}
		res := GenerateContentObject(schema, valueResolver, nil)

		expected := `{"age":21,"name":{"first":"Jane","last":"Doe"}}`
		resJs, _ := json.Marshal(res)
		assert.Equal(expected, string(resJs))
	})

	t.Run("with-no-properties", func(t *testing.T) {
		schema := CreateSchemaFromString(t, `{"type": "object"}`)
		res := GenerateContentObject(schema, nil, nil)
		assert.Nil(res)
	})

	t.Run("with-no-resolved-values", func(t *testing.T) {
		schema := CreateSchemaFromString(t, `
        {
            "type":"object",
            "properties": {
                "name": {
                    "type": "object",
                    "properties": {
                        "first": {"type": "string"}
                    }
                }
            }
        }`)
		expected := map[string]any{
			"name": map[string]any{},
		}
		res := GenerateContentObject(schema, nil, nil)
		assert.Equal(expected, res)
	})

	t.Run("with-additional-properties", func(t *testing.T) {
		valueReplacer := func(schema any, state *replacers.ReplaceState) any {
			return state.NamePath[0] + "-value"
		}
		schema := CreateSchemaFromString(t, `
		{
			"type": "object",
			"properties": {
				"name": {"type": "string"},
				"address": {"type": "string"}
			},
			"additionalProperties": {
				"type": "string"
			}
		}`)

		expected := map[string]any{
			"name":    "name-value",
			"address": "address-value",
			"extra-1": "extra-1-value",
			"extra-2": "extra-2-value",
			"extra-3": "extra-3-value",
		}

		res := GenerateContentObject(schema, valueReplacer, nil)
		assert.Equal(expected, res)
	})

	t.Run("with-max-properties", func(t *testing.T) {
		valueReplacer := func(schema any, state *replacers.ReplaceState) any {
			return state.NamePath[0] + "-value"
		}
		schema := CreateSchemaFromString(t, `
		{
			"type": "object",
 			"maxProperties": 1,
			"properties": {
				"name": {"type": "string"},
				"address": {"type": "string"}
			}
		}`)

		res := GenerateContentObject(schema, valueReplacer, nil)
		assert.Equal(1, len(res.(map[string]any)))
	})
}

func TestGenerateContentArray(t *testing.T) {
	assert := assert2.New(t)

	t.Run("generate simple array without min/max items", func(t *testing.T) {
		schema := CreateSchemaFromString(t, `{
            "type": "array",
            "items": {
                "type": "string"
            }
        }`)

		valueResolver := func(schema any, state *replacers.ReplaceState) any {
			return "foo"
		}

		res := GenerateContentArray(schema, valueResolver, nil)
		assert.ElementsMatch([]string{"foo"}, res)
	})

	t.Run("generate simple array", func(t *testing.T) {
		schema := CreateSchemaFromString(t, `{
            "type": "array",
			"minItems": 3,
            "items": {
                "type": "string"
            }
        }`)

		callNum := -1

		valueResolver := func(schema any, state *replacers.ReplaceState) any {
			callNum++
			items := []string{"a", "b", "c", "d"}
			return items[callNum]
		}

		res := GenerateContentArray(schema, valueResolver, nil)
		assert.ElementsMatch([]string{"a", "b", "c"}, res)
	})

	t.Run("with-no-resolved-values", func(t *testing.T) {
		schema := CreateSchemaFromString(t, `{
            "type": "array",
			"minItems": 3,
            "items": {"type": "string"}
        }`)
		res := GenerateContentArray(schema, nil, nil)
		assert.Nil(res)
	})
}

func TestGenerateRequestHeaders(t *testing.T) {
	assert := assert2.New(t)

	t.Run("GenerateRequestHeaders", func(t *testing.T) {
		valueResolver := func(schema any, state *replacers.ReplaceState) any {
			switch state.NamePath[len(state.NamePath)-1] {
			case "mode":
				return "dark"
			case "lang":
				return "de"
			case "x-key":
				return "abcdef"
			case "version":
				return "1.0.0"
			}
			return nil
		}
		params := openapi.Parameters{
			nil,
			newOpenAPIParameter("X-Key", openapi.ParameterInHeader, CreateSchemaFromString(t, `{"type": "string"}`)),
			newOpenAPIParameter("Version", openapi.ParameterInHeader, CreateSchemaFromString(t, `{"type": "string"}`)),
			newOpenAPIParameter("Preferences", openapi.ParameterInHeader, CreateSchemaFromString(t, `{"type": "object", "properties": {"mode": {"type": "string"}, "lang": {"type": "string"}}}`)),
			newOpenAPIParameter("id", openapi.ParameterInHeader, CreateSchemaFromString(t, `{"type": "string"}`)),
		}

		expected := map[string]any{
			"x-key":       "abcdef",
			"version":     "1.0.0",
			"preferences": map[string]any{"mode": "dark", "lang": "de"},
			"id":          nil,
		}

		res := GenerateRequestHeaders(params, valueResolver)
		assert.Equal(expected, res)
	})

	t.Run("param-is-nil", func(t *testing.T) {
		params := openapi.Parameters{{}}
		res := GenerateRequestHeaders(params, nil)
		assert.Nil(res)
	})

	t.Run("schema-is-nil", func(t *testing.T) {
		params := openapi.Parameters{newOpenAPIParameter("", openapi.ParameterInHeader, nil)}
		res := GenerateRequestHeaders(params, nil)
		assert.Nil(res)
	})

	t.Run("schema-is-nil", func(t *testing.T) {
		params := openapi.Parameters{
			newOpenAPIParameter("", openapi.ParameterInHeader, nil),
		}
		res := GenerateRequestHeaders(params, nil)
		assert.Nil(res)
	})
}

func TestGenerateResponseHeaders(t *testing.T) {
	assert := assert2.New(t)

	t.Run("GenerateResponseHeaders", func(t *testing.T) {
		valueReplacer := func(schema any, state *replacers.ReplaceState) any {
			switch state.NamePath[len(state.NamePath)-1] {
			case "x-rate-limit-limit":
				return 100
			case "x-rate-limit-remaining":
				return 80
			}
			return nil
		}
		headers := openapi.Headers{
			"X-Rate-Limit-Limit":     newOpenAPIParameter("X-Key", openapi.ParameterInHeader, CreateSchemaFromString(t, `{"type": "integer"}`)),
			"X-Rate-Limit-Remaining": newOpenAPIParameter("X-Key", openapi.ParameterInHeader, CreateSchemaFromString(t, `{"type": "integer"}`)),
		}

		expected := http.Header{
			"X-Rate-Limit-Limit":     []string{"100"},
			"X-Rate-Limit-Remaining": []string{"80"},
		}

		res := GenerateResponseHeaders(headers, valueReplacer)
		assert.Equal(expected, res)
	})
}

func TestGenerateContentFromJSON(t *testing.T) {
	assert := assert2.New(t)
	valueReplacer := func(schema any, state *replacers.ReplaceState) any {
		switch state.NamePath[len(state.NamePath)-1] {
		case "id":
			return 123
		case "name":
			return "Jane Doe"
		}
		return nil
	}

	t.Run("json-map-any-any", func(t *testing.T) {
		content := map[any]any{
			"id":   "{id}",
			"name": "{name}",
		}
		expected := map[any]any{
			"id":   123,
			"name": "Jane Doe",
		}

		res := generateContentFromJSON(content, valueReplacer, nil)
		assert.Equal(expected, res)
	})

	t.Run("json-map-string-any", func(t *testing.T) {
		content := map[string]any{
			"id":   "{id}",
			"name": "{name}",
		}
		expected := map[string]any{
			"id":   123,
			"name": "Jane Doe",
		}

		res := generateContentFromJSON(content, valueReplacer, nil)
		assert.Equal(expected, res)
	})

	t.Run("json-map-string-any-glued", func(t *testing.T) {
		content := map[string]any{
			"id":           "{id}",
			"name":         "{name}",
			"id-with-name": "{id}-{name}",
		}
		expected := map[string]any{
			"id":           123,
			"name":         "Jane Doe",
			"id-with-name": "123-Jane Doe",
		}

		res := generateContentFromJSON(content, valueReplacer, nil)
		assert.Equal(expected, res)
	})
	t.Run("json-slice-any", func(t *testing.T) {
		content := []any{"{id}", "{name}"}
		expected := []any{123, "Jane Doe"}

		res := generateContentFromJSON(content, valueReplacer, nil)
		assert.Equal(expected, res)
	})

	t.Run("json-unknown-underlying-struct", func(t *testing.T) {
		content := []string{"{name}", "{name}"}
		expected := []string{"{name}", "{name}"}

		res := generateContentFromJSON(content, valueReplacer, nil)
		assert.Equal(expected, res)
	})

	t.Run("no-placeholders", func(t *testing.T) {
		content := map[string]any{
			"id":   1234,
			"name": "John Doe",
		}

		res := generateContentFromJSON(content, valueReplacer, nil)
		assert.Equal(content, res)
	})
}
