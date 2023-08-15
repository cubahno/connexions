package xs

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func CreateDocumentFromString(t *testing.T, src string) *Document {
	doc, err := NewDocumentFromString(src)
	if err != nil {
		t.Errorf("Error loading document: %v", err)
		t.FailNow()
	}
	return doc
}

func CreateSchemaFromString(t *testing.T, src string) *Schema {
	schema := &Schema{}
	err := json.Unmarshal([]byte(src), schema)
	if err != nil {
		t.Errorf("Error parsing JSON: %v", err)
		t.FailNow()
	}
	return schema
}

func CreateOperationFromString(t *testing.T, src string) *Operation {
	res := &Operation{}
	err := json.Unmarshal([]byte(src), res)
	if err != nil {
		t.Errorf("Error parsing JSON: %v", err)
		t.FailNow()
	}
	return res
}

func AssertJSONEqual(t *testing.T, expected, actual any) {
	expectedJSON, _ := json.Marshal(expected)
	actualJSON, _ := json.Marshal(actual)

	// Compare JSON representations
	assert.Equal(t, string(expectedJSON), string(actualJSON), "JSON representations should match")
}
