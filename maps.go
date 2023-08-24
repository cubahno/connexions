package connexions

import (
	"math/rand"
	"reflect"
	"sort"
	"strings"
	"time"
)

func IsMap(i any) bool {
	val := reflect.ValueOf(i)
	return val.Kind() == reflect.Map
}

func GetValueByDottedPath(data map[string]any, path string) any {
	keys := strings.Split(path, ".")

	var current any = data

	for _, key := range keys {
		if val, ok := current.(map[string]any); ok {
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

func SetValueByDottedPath(data map[string]any, path string, value any) {
	keys := strings.Split(path, ".")
	lastIndex := len(keys) - 1

	var currentMap = data

	for i, key := range keys {
		if i == lastIndex {
			currentMap[key] = value
			break
		}

		if val, ok := currentMap[key]; ok {
			if nestedMap, nestedOk := val.(map[string]any); nestedOk {
				currentMap = nestedMap
			} else {
				return
			}
		} else {
			nestedMap := make(map[string]any)
			currentMap[key] = nestedMap
			currentMap = nestedMap
		}
	}
}

func GetRandomKeyFromMap[T any](m map[string]T) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	if len(keys) == 0 {
		return ""
	}

	rand.New(rand.NewSource(time.Now().UnixNano()))
	randomIndex := rand.Intn(len(keys))

	return keys[randomIndex]
}

func GetSortedMapKeys[T any](content map[string]T) []string {
	var keys []string
	for key := range content {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
