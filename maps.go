package xs

import (
	"fmt"
	"strings"
)

func GetValueByDottedPath(data map[string]any, path string) any {
	keys := strings.Split(path, ".")

	var current any = data

	for _, key := range keys {
		if val, ok := current.(map[string]interface{}); ok {
			if nestedVal, nestedOk := val[key]; nestedOk {
				current = nestedVal
			} else {
				return nil
			}
		} else {
			return nil
		}
	}

	return current
}

func SetValueByDottedPath(data map[string]interface{}, path string, value interface{}) {
	keys := strings.Split(path, ".")
	lastIndex := len(keys) - 1

	var currentMap = data

	for i, key := range keys {
		if i == lastIndex {
			currentMap[key] = value
			break
		}

		if val, ok := currentMap[key]; ok {
			if nestedMap, nestedOk := val.(map[string]interface{}); nestedOk {
				currentMap = nestedMap
			} else {
				fmt.Println("Invalid path:", path)
				return
			}
		} else {
			nestedMap := make(map[string]interface{})
			currentMap[key] = nestedMap
			currentMap = nestedMap
		}
	}
}
