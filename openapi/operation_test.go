package openapi

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	assert2 "github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"strings"
	"testing"
)

func TestEncodeContent(t *testing.T) {
	assert := assert2.New(t)

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

		result, err := EncodeContent(structContent, "application/x-yaml")
		assert.NoError(err)

		expectedYAML, err := yaml.Marshal(structContent)
		assert.NoError(err)

		assert.Equal(string(expectedYAML), string(result))
	})

	t.Run("Byte Content", func(t *testing.T) {
		content := []byte("hallo, welt!")
		result, err := EncodeContent(content, "x-unknown")
		assert.NoError(err)
		assert.Equal(content, result)
	})

	t.Run("string-content", func(t *testing.T) {
		content := "hallo, welt!"
		result, err := EncodeContent(content, "x-unknown")
		assert.NoError(err)
		assert.Equal(content, string(result))
	})

	t.Run("Unknown Content Type", func(t *testing.T) {
		content := 123
		result, err := EncodeContent(content, "x-unknown")
		assert.NoError(err)
		assert.Nil(result)
	})
}

func TestCreateCURLBody(t *testing.T) {
	assert := assert2.New(t)

	t.Run("nil-content", func(t *testing.T) {
		res, err := CreateCURLBody(nil, "application/json")
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

		result, err := CreateCURLBody(content, "application/x-www-form-urlencoded")
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

		result, err := CreateCURLBody(content, "application/x-www-form-urlencoded")
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

		result, err := CreateCURLBody(content, "multipart/form-data")
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

		result, err := CreateCURLBody(content, "multipart/form-data")
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

		result, err := CreateCURLBody(content, "application/json")
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

		result, err := CreateCURLBody(content, "application/xml")
		assert.NoError(err)

		enc, _ := xml.Marshal(content)
		expected := fmt.Sprintf("--data '%s'", string(enc))
		assert.Equal(expected, result)
	})

	t.Run("XML-invalid", func(t *testing.T) {
		t.Parallel()

		result, err := CreateCURLBody(func() {}, "application/xml")
		assert.Equal("", result)
		assert.Error(err)
	})

	t.Run("unknown-content-type", func(t *testing.T) {
		t.Parallel()

		result, err := CreateCURLBody(123, "application/unknown")
		assert.Equal("", result)
		assert.NoError(err)
	})
}
