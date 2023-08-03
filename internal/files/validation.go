package files

import (
	"encoding/json"
	"os"

	"go.yaml.in/yaml/v4"
)

// IsEmptyDir checks if a directory is empty.
func IsEmptyDir(path string) bool {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}
	return len(entries) == 0
}

// IsJsonType checks if the content is a valid JSON document.
func IsJsonType(content []byte) bool {
	var jsonData map[string]interface{}
	return json.Unmarshal(content, &jsonData) == nil
}

// IsYamlType checks if the content is a valid YAML document.
func IsYamlType(content []byte) bool {
	var yamlData map[string]interface{}
	err := yaml.Unmarshal(content, &yamlData)
	return err == nil
}
