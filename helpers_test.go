package connexions

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"testing"
)

func CreateSchemaFromYAMLFile(t *testing.T, filePath string, target any) {
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

	return
}

func CreateOperationFromYAMLFile(t *testing.T, filePath string, target any) {
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

func CreateKinSchemaFromString(t *testing.T, src string) *openapi3.Schema {
	schema := openapi3.NewSchema()
	err := json.Unmarshal([]byte(src), schema)
	if err != nil {
		t.Errorf("Error parsing JSON: %v", err)
		t.FailNow()
	}
	return schema
}

func CreateSchemaFromString(t *testing.T, src string) *Schema {
	return NewSchemaFromKin(CreateKinSchemaFromString(t, src), nil)
}

func AssertJSONEqual(t *testing.T, expected, actual any) {
	expectedJSON, _ := json.Marshal(expected)
	actualJSON, _ := json.Marshal(actual)

	// Compare JSON representations
	assert.Equal(t, string(expectedJSON), string(actualJSON), "JSON representations should match")
}

func GetJSONPair(expected, actual any) (string, string) {
	expectedJSON, _ := json.Marshal(expected)
	actualJSON, _ := json.Marshal(actual)

	return string(expectedJSON), string(actualJSON)
}

func SetupApp(appDir string) (*Router, error) {
	err := os.MkdirAll(filepath.Join(appDir, "resources"), 0755)
	if err != nil {
		return nil, err
	}

	err = os.Mkdir(filepath.Join(appDir, "resources", "contexts"), 0755)
	if err != nil {
		return nil, err
	}
	err = os.Mkdir(filepath.Join(appDir, "resources", "services"), 0755)
	if err != nil {
		return nil, err
	}

	err = CopyFile(filepath.Join("resources", "config.yml.dist"), filepath.Join(appDir, "resources", "config.yml.dist"))
	if err != nil {
		return nil, err
	}
	err = CopyFile(filepath.Join("resources", "config.yml.dist"), filepath.Join(appDir, "resources", "config.yml"))
	if err != nil {
		return nil, err
	}

	cfg := MustConfig(appDir)

	err = MustFileStructure(cfg.App.Paths)
	if err != nil {
		return nil, err
	}

	return NewRouter(cfg), nil
}

func UnmarshallResponse[T any](t *testing.T, res *bytes.Buffer) *T {
	target := new(T)
	err := json.Unmarshal(res.Bytes(), &target)
	if err != nil {
		t.Errorf("Error unmarshaling JSON: %v\n", err)
		t.FailNow()
	}
	return target
}

func AddTestFileToForm(writer *multipart.Writer, fieldName, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	baseName := filepath.Base(filePath)

	part, err := writer.CreateFormFile(fieldName, baseName)
	if err != nil {
		return err
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return err
	}

	return nil
}

func CreateTestZip(files map[string]string) *bytes.Buffer {
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	if len(files) == 0 {
		files = map[string]string{
			"file1.txt": "This is file 1 content.",
			"file2.txt": "This is file 2 content.",
		}
	}

	for name, contents := range files {
		file, _ := zipWriter.Create(name)
		file.Write([]byte(contents))
	}

	zipWriter.Close()
	return &buf
}

func CreateTestMapFormReader(data map[string]string) (*multipart.Writer, *bytes.Buffer) {
	var bodyBuffer bytes.Buffer
	writer := multipart.NewWriter(&bodyBuffer)

	for k, v := range data {
		err := writer.WriteField(k, v)
		if err != nil {
			return nil, nil
		}
	}

	writer.Close()

	return writer, &bodyBuffer
}
