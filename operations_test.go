package connexions

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func NewOpenAPIParameter(name, in string, schema *Schema) *OpenAPIParameter {
	return &OpenAPIParameter{
		Name:   name,
		In:     in,
		Schema: schema,
	}
}

func CreateKinDocumentFromFile(t *testing.T, filePath string) Document {
	doc, err := NewKinDocumentFromFile(filePath)
	if err != nil {
		t.Errorf("Error loading document: %v", err)
		t.FailNow()
	}
	return doc
}

func CreateSchemaFromFile(t *testing.T, filePath string) *Schema {
	cont, err := os.ReadFile(filePath)
	if err != nil {
		t.Errorf("Error reading file: %v", err)
		t.FailNow()
	}

	// remove schema key if pre
	tmp := make(map[string]any)
	_ = json.Unmarshal(cont, &tmp)
	if _, ok := tmp["schema"]; ok {
		cont, _ = json.Marshal(tmp["schema"])
	}

	return CreateSchemaFromString(t, string(cont))
}

func TestNewRequestFromOperation(t *testing.T) {
	t.Run("base-case", func(t *testing.T) {
		valueResolver := func(content any, state *ReplaceState) any {
			schema, _ := content.(*Schema)
			if state.NamePath[0] == "userId" {
				return "123"
			}
			if schema.Example != nil {
				return schema.Example
			}
			return schema.Default
		}

		operation := CreateKinOperationFromFile(t, filepath.Join(TestSchemaPath, "operation.json"))
		req := NewRequestFromOperation("/foo", "/users/{userId}", "POST", operation, valueResolver)

		expectedBodyM := map[string]any{
			"username": "john_doe",
			"email":    "john.doe@example.com",
		}
		expectedBodyB, _ := json.Marshal(expectedBodyM)

		expectedHeaders := map[string]any{"lang": "de"}

		assert.Equal(t, "POST", req.Method)
		assert.Equal(t, "/foo/users/123", req.Path)
		assert.Equal(t, "limit=10", req.Query)
		assert.Equal(t, "application/json", req.ContentType)
		assert.Equal(t, string(expectedBodyB), req.Body)
		assert.Equal(t, expectedHeaders, req.Headers)
	})
}

func TestEncodeContent(t *testing.T) {
	t.Run("Nil Content", func(t *testing.T) {
		result, err := EncodeContent(nil, "application/json")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result != nil {
			t.Errorf("Expected empty string, got: %s", result)
		}
	})

	t.Run("String Content", func(t *testing.T) {
		content := "Hello, world!"
		result, err := EncodeContent(content, "application/json")
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
		result, err := EncodeContent(content, "application/json")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if string(result) != expectedResult {
			t.Errorf("Expected '%s', got: %s", expectedResult, result)
		}
	})

	t.Run("XML Content", func(t *testing.T) {
		ass := assert.New(t)
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

		result, err := EncodeContent(structContent, "application/xml")
		ass.NoError(err)

		expectedXML, err := xml.Marshal(structContent)
		ass.NoError(err)

		ass.Equal(string(expectedXML), string(result))
	})
}

func TestCreateCURLBody(t *testing.T) {
	ass := assert.New(t)

	t.Run("FormURLEncoded", func(t *testing.T) {
		t.Parallel()

		content := map[string]interface{}{
			"name":  "John",
			"age":   30,
			"email": "john@example.com",
		}

		result, err := CreateCURLBody(content, "application/x-www-form-urlencoded")
		ass.NoError(err)

		expected := `--data-urlencode 'age=30' \
--data-urlencode 'email=john%40example.com' \
--data-urlencode 'name=John'
`
		expected = strings.TrimSuffix(expected, "\n")
		ass.Equal(expected, result)
	})

	t.Run("MultipartFormData", func(t *testing.T) {
		t.Parallel()

		content := map[string]interface{}{
			"name":  "Jane",
			"age":   25,
			"email": "jane@example.com",
		}

		result, err := CreateCURLBody(content, "multipart/form-data")
		ass.NoError(err)

		expected := `--form 'age="25"' \
--form 'email="jane%40example.com"' \
--form 'name="Jane"'
`
		expected = strings.TrimSuffix(expected, "\n")
		ass.Equal(expected, result)
	})

	t.Run("JSON", func(t *testing.T) {
		t.Parallel()

		content := map[string]interface{}{
			"name":  "Alice",
			"age":   28,
			"email": "alice@example.com",
		}

		result, err := CreateCURLBody(content, "application/json")
		ass.NoError(err)

		enc, _ := json.Marshal(content)
		expected := fmt.Sprintf("--data-raw '%s'", string(enc))
		ass.Equal(expected, result)
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

		result, err := CreateCURLBody(content, "application/xml")
		ass.NoError(err)

		enc, _ := xml.Marshal(content)
		expected := fmt.Sprintf("--data '%s'", string(enc))
		ass.Equal(expected, result)
	})
}

func TestNewResponse(t *testing.T) {
	t.Run("base-case", func(t *testing.T) {
		valueResolver := func(content any, state *ReplaceState) any {
			schema, _ := content.(*Schema)
			if state.NamePath[0] == "userId" {
				return 123
			}
			if schema.Example != nil {
				return schema.Example
			}
			return schema.Default
		}

		operation := CreateKinOperationFromFile(t, filepath.Join(TestSchemaPath, "operation-base.json"))
		res := NewResponseFromOperation(operation, valueResolver)

		expectedHeaders := http.Header{
			"Location":     []string{"https://example.com/users/123"},
			"Content-Type": []string{"application/json"},
		}
		expectedContentM := map[string]any{
			"id":    float64(123),
			"email": "jane.doe@example.com",
		}
		expectedContent, _ := json.Marshal(expectedContentM)

		assert.Equal(t, "application/json", res.ContentType)
		assert.Equal(t, 200, res.StatusCode)
		assert.Equal(t, expectedHeaders, res.Headers)
		assert.Equal(t, expectedContent, res.Content)
	})

	t.Run("no-content-type", func(t *testing.T) {
		valueResolver := func(content any, state *ReplaceState) any {
			schema, _ := content.(*Schema)
			if state.NamePath[0] == "userId" {
				return 123
			}
			if schema.Example != nil {
				return schema.Example
			}
			return schema.Default
		}

		operation := CreateKinOperationFromFile(t, filepath.Join(TestSchemaPath, "operation-without-content-type.json"))
		res := NewResponseFromOperation(operation, valueResolver)

		expectedHeaders := http.Header{
			"Content-Type": []string{"text/plain"},
			"Location":     []string{"https://example.com/users/123"},
		}

		assert.Equal(t, 200, res.StatusCode)
		assert.Equal(t, expectedHeaders, res.Headers)

		assert.Equal(t, "text/plain", res.ContentType)
		assert.Nil(t, res.Content)
	})
}

func TestTransformHTTPCode(t *testing.T) {
	type tc struct {
		name     string
		expected int
	}
	testCases := []tc{
		{"200", 200},
		{"2xx", 200},
		{"2XX", 200},
		{"default", 200},
		{"20x", 200},
		{"201", 201},
		{"*", 200},
		{"unknown", 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, TransformHTTPCode(tc.name))
		})
	}
}

func TestGenerateURL(t *testing.T) {
	t.Run("params correctly replaced in path", func(t *testing.T) {
		path := "/users/{id}/{file-id}"
		valueResolver := func(content any, state *ReplaceState) any {
			if state.NamePath[0] == "id" {
				return 123
			}
			if state.NamePath[0] == "file-id" {
				return "foo"
			}
			return "something-else"
		}
		params := OpenAPIParameters{
			NewOpenAPIParameter("id", "path", CreateSchemaFromString(t, `{"type": "integer"}`)),
			NewOpenAPIParameter("file-id", "path", CreateSchemaFromString(t, `{"type": "string"}`)),
			NewOpenAPIParameter("file-id", "query", CreateSchemaFromString(t, `{"type": "integer"}`)),
		}
		res := GenerateURLFromSchemaParameters(path, valueResolver, params)
		assert.Equal(t, "/users/123/foo", res)
	})
}

func TestGenerateQuery(t *testing.T) {
	t.Run("params correctly replaced in query", func(t *testing.T) {
		valueResolver := func(schema any, state *ReplaceState) any {
			if state.NamePath[0] == "id" {
				return 123
			}
			if state.NamePath[0] == "file-id" {
				return "foo"
			}
			return "something-else"
		}
		params := OpenAPIParameters{
			NewOpenAPIParameter("id", "query", CreateSchemaFromString(t, `{"type": "integer"}`)),
			NewOpenAPIParameter("file-id", "query", CreateSchemaFromString(t, `{"type": "foo"}`)),
		}
		res := GenerateQuery(valueResolver, params)

		// TODO(igor): fix order of query params
		assert.Contains(t, []string{"id=123&file-id=foo", "file-id=foo&id=123"}, res)
	})

	t.Run("arrays in url", func(t *testing.T) {
		valueResolver := func(content any, state *ReplaceState) any {
			return "foo bar"
		}
		params := OpenAPIParameters{
			NewOpenAPIParameter(
				"tags",
				"query",
				CreateSchemaFromString(t, `{"type": "array", "items": {"type": "string"}}`),
			),
		}
		res := GenerateQuery(valueResolver, params)

		expected := "tags[]=foo+bar&tags[]=foo+bar"
		assert.Equal(t, expected, res)
	})

	t.Run("no-resolved-values", func(t *testing.T) {
		valueResolver := func(schema any, state *ReplaceState) any {
			return nil
		}
		params := OpenAPIParameters{
			NewOpenAPIParameter(
				"id",
				"query",
				CreateSchemaFromString(t, `{"type": "integer"}`),
			),
		}
		res := GenerateQuery(valueResolver, params)

		expected := "id="
		assert.Equal(t, expected, res)
	})
}

func TestGenerateContent(t *testing.T) {
	t.Run("base-case", func(t *testing.T) {
		valueResolver := func(content any, state *ReplaceState) any {
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
		schema := CreateSchemaFromFile(t, filepath.Join(TestSchemaPath, "schema-base.json"))
		res := GenerateContentFromSchema(schema, valueResolver, nil)

		expected := map[string]any{
			"user": map[string]any{"id": 21, "score": 11.5},
			"pages": []any{
				map[string]any{
					"limit": 100, "tag1": "#dice", "tag2": "#nice", "offset": -1, "first": 10,
				},
				map[string]any{
					"limit": 100, "tag1": "#dice", "tag2": "#nice", "offset": -1, "first": 10,
				},
			},
		}
		assert.Equal(t, expected, res)
	})

	t.Run("with-nested-all-of", func(t *testing.T) {
		valueResolver := func(schema any, state *ReplaceState) any {
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

		schema := CreateSchemaFromFile(t, filepath.Join(TestSchemaPath, "schema-with-nested-all-of.json"))
		expected := map[string]any{"name": "Jane Doe", "age": 30, "tag": "#doe", "league": "premier", "rating": 345.6}

		res := GenerateContentFromSchema(schema, valueResolver, nil)
		assert.Equal(t, expected, res)
	})

	t.Run("fast-track-used-with-object", func(t *testing.T) {
		dice := map[string]string{"nice": "very nice", "rice": "good rice"}

		valueResolver := func(schema any, state *ReplaceState) any {
			switch state.NamePath[0] {
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

		expected := map[string]any{"dice": dice}
		assert.Equal(t, expected, res)
	})

	t.Run("with-circular-array-references", func(t *testing.T) {
		valueResolver := func(schema any, state *ReplaceState) any {
			switch state.NamePath[len(state.NamePath)-1] {
			case "id":
				return 123
			case "name":
				return "noda-123"
			}
			return nil
		}
		doc := CreateKinDocumentFromFile(t, filepath.Join(TestSchemaPath, "doc-with-circular-array.json"))
		resp, _ := doc.FindOperation("/nodes/{id}", http.MethodGet).GetResponse()
		schema, _ := resp.GetContent()
		res := GenerateContentFromSchema(schema, valueResolver, nil)

		expected := map[string]any{
			"id":   123,
			"name": "noda-123",
			"children": []any{
				map[string]any{
					"id":       123,
					"name":     "noda-123",
					"children": nil,
				},
				map[string]any{
					"id":       123,
					"name":     "noda-123",
					"children": nil,
				},
			},
		}
		assert.Equal(t, expected, res)
	})

	t.Run("with-circular-object-references", func(t *testing.T) {
		valueResolver := func(schema any, state *ReplaceState) any {
			switch state.NamePath[len(state.NamePath)-1] {
			case "id":
				return 123
			case "name":
				return "noda-123"
			}
			return nil
		}
		filePath := filepath.Join(TestSchemaPath, "circular-with-references.json")
		doc := CreateKinDocumentFromFile(t, filePath)
		resp, _ := doc.FindOperation("/nodes/{id}", http.MethodGet).GetResponse()
		schema, _ := resp.GetContent()
		res := GenerateContentFromSchema(schema, valueResolver, nil)

		expected := map[string]any{
			"id":   123,
			"name": "noda-123",
			"parent": map[string]any{
				"id":   123,
				"name": "noda-123",
			},
		}
		assert.Equal(t, expected, res)
	})

	t.Run("with-circular-object-references-inlined", func(t *testing.T) {
		valueResolver := func(schema any, state *ReplaceState) any {
			switch state.NamePath[len(state.NamePath)-1] {
			case "id":
				return 123
			case "name":
				return "noda-123"
			}
			return nil
		}
		filePath := filepath.Join(TestSchemaPath, "circular-with-inline.json")
		doc := CreateKinDocumentFromFile(t, filePath)
		resp, _ := doc.FindOperation("/nodes/{id}", http.MethodGet).GetResponse()
		schema, _ := resp.GetContent()
		res := GenerateContentFromSchema(schema, valueResolver, nil)

		expected := map[string]any{
			"id":   123,
			"name": "noda-123",
			"parent": map[string]any{
				"id":     123,
				"name":   "noda-123",
				"parent": nil,
			},
		}
		assert.Equal(t, expected, res)
	})
}

func TestGenerateContentObject(t *testing.T) {
	t.Run("GenerateContentObject", func(t *testing.T) {
		schema := CreateSchemaFromFile(t, filepath.Join(TestSchemaPath, "schema-with-name-obj-and-age.json"))

		valueResolver := func(schema any, state *ReplaceState) any {
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
		assert.Equal(t, expected, string(resJs))
	})

	t.Run("with-no-properties", func(t *testing.T) {
		schema := CreateSchemaFromString(t, `{"type": "object"}`)
		res := GenerateContentObject(schema, nil, nil)
		assert.Nil(t, res)
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
			"name": map[string]any{
				"first": nil,
			},
		}
		res := GenerateContentObject(schema, nil, nil)
		assert.Equal(t, expected, res)
	})
}

func TestGenerateContentArray(t *testing.T) {
	t.Run("generate simple array without min/max items", func(t *testing.T) {
		schema := CreateSchemaFromString(t, `{
            "type": "array",
            "items": {
                "type": "string"
            }
        }`)

		valueResolver := func(schema any, state *ReplaceState) any {
			return "foo"
		}

		res := GenerateContentArray(schema, valueResolver, nil)
		assert.ElementsMatch(t, []string{"foo", "foo"}, res)
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

		valueResolver := func(schema any, state *ReplaceState) any {
			callNum++
			items := []string{"a", "b", "c", "d"}
			return items[callNum]
		}

		res := GenerateContentArray(schema, valueResolver, nil)
		assert.ElementsMatch(t, []string{"a", "b", "c", "d"}, res)
	})

	t.Run("with-no-resolved-values", func(t *testing.T) {
		schema := CreateSchemaFromString(t, `{
            "type": "array",
			"minItems": 3,
            "items": {"type": "string"}
        }`)
		res := GenerateContentArray(schema, nil, nil)
		assert.Nil(t, res)
	})
}

func TestGenerateRequestHeaders(t *testing.T) {
	t.Run("GenerateRequestHeaders", func(t *testing.T) {
		valueResolver := func(schema any, state *ReplaceState) any {
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
		params := OpenAPIParameters{
			NewOpenAPIParameter("X-Key", ParameterInHeader, CreateSchemaFromString(t, `{"type": "string"}`)),
			NewOpenAPIParameter("Version", ParameterInHeader, CreateSchemaFromString(t, `{"type": "string"}`)),
			NewOpenAPIParameter("Preferences", ParameterInHeader, CreateSchemaFromString(t, `{"type": "object", "properties": {"mode": {"type": "string"}, "lang": {"type": "string"}}}`)),
			NewOpenAPIParameter("id", ParameterInHeader, CreateSchemaFromString(t, `{"type": "string"}`)),
		}

		expected := map[string]any{
			"x-key":       "abcdef",
			"version":     "1.0.0",
			"preferences": map[string]any{"mode": "dark", "lang": "de"},
			"id":          nil,
		}

		res := GenerateRequestHeaders(params, valueResolver)
		assert.Equal(t, expected, res)
	})

	t.Run("param-is-nil", func(t *testing.T) {
		params := OpenAPIParameters{{}}
		res := GenerateRequestHeaders(params, nil)
		assert.Nil(t, res)
	})

	t.Run("schema-is-nil", func(t *testing.T) {
		params := OpenAPIParameters{NewOpenAPIParameter("", ParameterInHeader, nil)}
		res := GenerateRequestHeaders(params, nil)
		assert.Nil(t, res)
	})

	t.Run("schema-is-nil", func(t *testing.T) {
		params := OpenAPIParameters{
			NewOpenAPIParameter("", ParameterInHeader, nil),
		}
		res := GenerateRequestHeaders(params, nil)
		assert.Nil(t, res)
	})
}

func TestGenerateResponseHeaders(t *testing.T) {
	t.Run("GenerateResponseHeaders", func(t *testing.T) {
		valueResolver := func(schema any, state *ReplaceState) any {
			switch state.NamePath[len(state.NamePath)-1] {
			case "x-rate-limit-limit":
				return 100
			case "x-rate-limit-remaining":
				return 80
			}
			return nil
		}
		headers := OpenAPIHeaders{
			"X-Rate-Limit-Limit":     NewOpenAPIParameter("X-Key", ParameterInHeader, CreateSchemaFromString(t, `{"type": "integer"}`)),
			"X-Rate-Limit-Remaining": NewOpenAPIParameter("X-Key", ParameterInHeader, CreateSchemaFromString(t, `{"type": "integer"}`)),
		}

		expected := http.Header{
			"X-Rate-Limit-Limit":     []string{"100"},
			"X-Rate-Limit-Remaining": []string{"80"},
		}

		res := GenerateResponseHeaders(headers, valueResolver)
		assert.Equal(t, expected, res)
	})
}
