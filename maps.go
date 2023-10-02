package connexions

import (
	"fmt"
	"math/rand"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"time"
)

// IsMap checks if the value is a map
func IsMap(i any) bool {
	val := reflect.ValueOf(i)
	return val.Kind() == reflect.Map
}

// GetValueByDottedPath returns the value of the given path in the given map.
// If the path does not exist, nil is returned.
// e.g. GetValueByDottedPath(map[string]any{"a": map[string]any{"b": 1}}, "a.b") returns 1
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

// SetValueByDottedPath sets the value of the given path in the given map.
// If the path does not exist, it is created.
// e.g. SetValueByDottedPath(map[string]any{"a": map[string]any{"b": 1}}, "a.b", 2) sets the value of "a.b" to 2
// ! This function modifies the given map.
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

// GetRandomKeyFromMap returns a random key from the given map.
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

// GetSortedMapKeys returns the keys of the given map sorted alphabetically.
func GetSortedMapKeys[T any](content map[string]T) []string {
	var keys []string
	for key := range content {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// CopyNestedMap returns a copy of the given map with all nested maps copied as well.
func CopyNestedMap(source map[string]map[string]any) map[string]map[string]any {
	res := make(map[string]map[string]any)
	for key, value := range source {
		copyValue := make(map[string]any)
		for innerKey, innerValue := range value {
			copyValue[innerKey] = innerValue
		}
		res[key] = copyValue
	}
	return res
}

// MapToURLEncodedForm converts a map to a URL encoded form.
// e.g. {"a": {"b": 1}} becomes "a[b]=1"
func MapToURLEncodedForm(data map[string]any) string {
	values := url.Values{}
	buildURLValues(data, "", values)
	return values.Encode()
}

func buildURLValues(data any, prefix string, values url.Values) {
	switch v := data.(type) {
	case map[string]interface{}:
		for key, value := range v {
			newKey := key
			if prefix != "" {
				newKey = prefix + "[" + key + "]"
			}
			buildURLValues(value, newKey, values)
		}
	case []interface{}:
		for i, item := range v {
			arrayKey := fmt.Sprintf("%s[%d]", prefix, i)
			buildURLValues(item, arrayKey, values)
		}
	default:
		values.Add(prefix, ToString(v))
	}
}
