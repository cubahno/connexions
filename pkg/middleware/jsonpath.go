package middleware

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// pathSegment represents a single segment in a dotted path.
type pathSegment struct {
	key   string
	index int  // -1 means no index
	isArr bool // true if this segment has an array index like [0]
}

// extractJSONPath extracts a value from JSON bytes using a dotted path.
// Supports:
//   - Simple: "data.name" - traverse nested objects
//   - Array index: "data.items[0].name" - specific element
//   - Array wildcard: "data.items.name" - when items is an array, search each element
//   - Top-level array: "[0].name" - index into a root-level array
func extractJSONPath(data []byte, path string) any {
	var parsed any
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil
	}

	segments := parseDottedPath(path)
	return navigatePath(parsed, segments)
}

// parseDottedPath splits a dotted path into segments.
// "data.items[0].name" → [{key:"data"}, {key:"items", index:0, isArr:true}, {key:"name"}]
// "[0].name"           → [{key:"", index:0, isArr:true}, {key:"name"}]
func parseDottedPath(path string) []pathSegment {
	parts := strings.Split(path, ".")
	segments := make([]pathSegment, 0, len(parts))

	for _, part := range parts {
		if part == "" {
			continue
		}

		if idx := strings.Index(part, "["); idx != -1 {
			key := part[:idx]
			indexStr := strings.TrimSuffix(part[idx+1:], "]")
			index, err := strconv.Atoi(indexStr)
			if err != nil {
				// Invalid index, treat as plain key
				segments = append(segments, pathSegment{key: part, index: -1})
				continue
			}
			segments = append(segments, pathSegment{key: key, index: index, isArr: true})
		} else {
			segments = append(segments, pathSegment{key: part, index: -1})
		}
	}

	return segments
}

// navigatePath traverses the parsed JSON structure following the path segments.
func navigatePath(current any, segments []pathSegment) any {
	for i, seg := range segments {
		if current == nil {
			return nil
		}

		// Handle array at current level (top-level or bare index after traversal)
		if arr, ok := current.([]any); ok {
			if seg.isArr && seg.key == "" {
				// Bare index like [0] - index directly into the current array
				if seg.index < 0 || seg.index >= len(arr) {
					return nil
				}
				current = arr[seg.index]
				continue
			}
			// Array wildcard - search each element with remaining segments
			remaining := segments[i:]
			for _, elem := range arr {
				result := navigatePath(elem, remaining)
				if result != nil {
					return result
				}
			}
			return nil
		}

		switch v := current.(type) {
		case map[string]any:
			val, ok := v[seg.key]
			if !ok {
				return nil
			}

			if seg.isArr {
				// Need to index into an array
				arr, ok := val.([]any)
				if !ok || seg.index < 0 || seg.index >= len(arr) {
					return nil
				}
				current = arr[seg.index]
			} else {
				// Check if val is an array and we have more segments - wildcard search
				if arr, ok := val.([]any); ok && i+1 < len(segments) {
					remaining := segments[i+1:]
					for _, elem := range arr {
						result := navigatePath(elem, remaining)
						if result != nil {
							return result
						}
					}
					return nil
				}
				current = val
			}

		default:
			return nil
		}
	}

	return current
}

// extractBodyValue extracts a field value from the request body.
// For form-encoded content type, parses as URL-encoded form data.
// Otherwise, parses as JSON using dotted path notation.
func extractBodyValue(body []byte, contentType string, field string) any {
	if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		params, err := url.ParseQuery(string(body))
		if err == nil {
			if v := params.Get(field); v != "" || params.Has(field) {
				return v
			}
		}
		return nil
	}
	return extractJSONPath(body, field)
}

// formatValue converts a value to a stable string representation for key building.
func formatValue(v any) string {
	if v == nil {
		return "<nil>"
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		// JSON numbers are float64; format integers without decimal
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	default:
		return fmt.Sprintf("%v", val)
	}
}
