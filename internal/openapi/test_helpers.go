package openapi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/cubahno/connexions/internal/types"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

var (
	_, b, _, _   = runtime.Caller(0)
	testDataPath = filepath.Join(filepath.Dir(b), "..", "", "..", "testdata")
)

func GetJSONPair(expected, actual any) (string, string) {
	expectedJSON, _ := json.Marshal(expected)
	actualJSON, _ := json.Marshal(actual)

	return string(expectedJSON), string(actualJSON)
}

func CreateSchemaFromYAMLFile(t *testing.T, filePath string, target any) {
	t.Helper()
	cont, err := os.ReadFile(filePath)
	if err != nil {
		t.Errorf("Error reading file: %v", err)
		t.FailNow()
	}

	// remove schema key if pre
	tmp := make(map[string]any)
	_ = yaml.Unmarshal(cont, &tmp)
	if _, ok := tmp["schema"]; ok {
		cont, _ = yaml.Marshal(tmp["schema"])
	}

	// to json
	var data any
	err = yaml.Unmarshal(cont, &data)
	if err != nil {
		t.Errorf("Error unmarshaling YAML: %v\n", err)
		t.FailNow()
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		t.Errorf("Error marshaling JSON: %v\n", err)
		t.FailNow()
	}

	// target := openapi3.NewSchema()
	err = json.Unmarshal(jsonBytes, target)
	if err != nil {
		t.Errorf("Error parsing JSON: %v", err)
		t.FailNow()
	}
}

func CreateOperationFromYAMLFile(t *testing.T, filePath string, target any) {
	t.Helper()
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

	// to json
	var data any
	err = yaml.Unmarshal(cont, &data)
	if err != nil {
		t.Errorf("Error unmarshaling YAML: %v\n", err)
		t.FailNow()
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		t.Errorf("Error marshaling JSON: %v\n", err)
		t.FailNow()
	}

	err = json.Unmarshal(jsonBytes, target)
	if err != nil {
		t.Errorf("Error parsing JSON: %v", err)
		t.FailNow()
	}
}

func CreateKinSchemaFromString(t *testing.T, jsonSrc string) *openapi3.Schema {
	t.Helper()
	schema := openapi3.NewSchema()
	err := json.Unmarshal([]byte(jsonSrc), schema)
	if err != nil {
		t.Errorf("Error parsing JSON: %v", err)
		t.FailNow()
	}
	return schema
}

func CreateSchemaFromString(t *testing.T, jsonSrc string) *types.Schema {
	t.Helper()
	return NewSchemaFromKin(CreateKinSchemaFromString(t, jsonSrc), nil)
}

func AssertJSONEqual(t *testing.T, expected, actual any) {
	t.Helper()
	expectedJSON, _ := json.Marshal(expected)
	actualJSON, _ := json.Marshal(actual)

	// Compare JSON representations
	assert.Equal(t, string(expectedJSON), string(actualJSON), "JSON representations should match")
}
