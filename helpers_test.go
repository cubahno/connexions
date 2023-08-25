package connexions

import (
	"encoding/json"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func CreateDocumentFromFile(t *testing.T, filePath string) Document {
	doc, err := NewKinDocumentFromFile(filePath)
	if err != nil {
		t.Errorf("Error loading document: %v", err)
		t.FailNow()
	}
	return doc
}

func CreateKinSchemaFromString(t *testing.T, src string) *openapi3.Schema {
	schema := openapi3.NewSchema()
	err := json.Unmarshal([]byte(src), schema)
	if err != nil {
		t.Errorf("Error parsing JSON: %v", err)
		t.FailNow()
	}
	return schema
}

func CreateKinSchemaFromFile(t *testing.T, filePath string) *openapi3.Schema {
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

	return CreateKinSchemaFromString(t, string(cont))
}

func CreateSchemaFromString(t *testing.T, src string) *Schema {
	return NewSchemaFromKin(CreateKinSchemaFromString(t, src), nil)
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

func CreateOperationFromString(t *testing.T, src string) Operationer {
	res := &KinOperation{Operation: openapi3.NewOperation()}
	err := json.Unmarshal([]byte(src), res)
	if err != nil {
		t.Errorf("Error parsing JSON: %v", err)
		t.FailNow()
	}
	return res
}

func CreateOperationFromFile(t *testing.T, filePath string) Operationer {
	cont, err := os.ReadFile(filePath)
	if err != nil {
		t.Errorf("Error reading file: %v", err)
		t.FailNow()
	}

	// remove schema key if pre
	tmp := make(map[string]any)
	_ = json.Unmarshal(cont, &tmp)
	if _, ok := tmp["operation"]; ok {
		cont, _ = json.Marshal(tmp["operation"])
	}

	return CreateOperationFromString(t, string(cont))
}

func NewOpenAPIParameter(name, in string, schema *Schema) *OpenAPIParameter {
	return &OpenAPIParameter{
		Name:   name,
		In:     in,
		Schema: schema,
	}
}

func AssertJSONEqual(t *testing.T, expected, actual any) {
	expectedJSON, _ := json.Marshal(expected)
	actualJSON, _ := json.Marshal(actual)

	// Compare JSON representations
	assert.Equal(t, string(expectedJSON), string(actualJSON), "JSON representations should match")
}
