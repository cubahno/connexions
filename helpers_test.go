package xs

import (
	"encoding/json"
	"github.com/getkin/kin-openapi/openapi3"
	"testing"
)

func CreateDocumentFromString(t *testing.T, src string) *openapi3.T {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData([]byte(src))
	if err != nil {
		t.Errorf("Error loading document: %v", err)
		t.FailNow()
	}
	return doc
}

func CreateSchemaFromString(t *testing.T, src string) *openapi3.Schema {
	schema := &openapi3.Schema{}
	err := json.Unmarshal([]byte(src), schema)
	if err != nil {
		t.Errorf("Error parsing JSON: %v", err)
		t.FailNow()
	}
	return schema
}

func CreateOperationFromString(t *testing.T, src string) *openapi3.Operation {
	res := &openapi3.Operation{}
	err := json.Unmarshal([]byte(src), res)
	if err != nil {
		t.Errorf("Error parsing JSON: %v", err)
		t.FailNow()
	}
	return res
}
