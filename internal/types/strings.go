package types

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap   = regexp.MustCompile("([a-z0-9])([A-Z])")
)

// ToSnakeCase converts a string to snake_case case.
// If the result starts with a digit, it prepends "n_" to make it a valid Go identifier.
var nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)

func ToSnakeCase(input string) string {
	snake := matchFirstCap.ReplaceAllString(input, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	snake = strings.ToLower(snake)
	snake = nonAlphaNum.ReplaceAllString(snake, "_")
	snake = strings.Trim(snake, "_")

	if len(snake) > 0 && snake[0] >= '0' && snake[0] <= '9' {
		snake = "n_" + snake
	}
	return snake
}

// MaybeRegexPattern checks if the input string contains any special characters.
// This is a simple good-enough check to see if the context key is a regex pattern.
func MaybeRegexPattern(input string) bool {
	specialChars := []string{"\\", ".", "*", "^", "$", "+", "?", "(", "[", "{", "|"}
	for _, char := range specialChars {
		if strings.Contains(input, char) {
			return true
		}
	}
	return false
}

func ToString(value any) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

// ExtractPlaceholders extracts all placeholders including curly brackets from a pattern.
func ExtractPlaceholders(input string) []string {
	return PlaceholderRegex.FindAllString(input, -1)
}

func Base64Encode(input string) string {
	return base64.StdEncoding.EncodeToString([]byte(input))
}

// DeduplicatePathParams renames duplicate path parameters in a path string.
// For example, "/foo/{id}/bar/{id}" becomes "/foo/{id}/bar/{id_2}".
// This is necessary because Chi router panics when registering routes with duplicate parameter names.
func DeduplicatePathParams(path string) string {
	seen := make(map[string]int)
	var result strings.Builder

	i := 0
	for i < len(path) {
		if path[i] == '{' {
			end := strings.Index(path[i:], "}")
			if end == -1 {
				result.WriteByte(path[i])
				i++
				continue
			}
			end += i + 1

			// extract name without braces
			name := path[i+1 : end-1]
			count := seen[name]
			seen[name] = count + 1

			if count > 0 {
				fmt.Fprintf(&result, "{%s_%d}", name, count+1)
			} else {
				result.WriteString(path[i:end])
			}
			i = end
		} else {
			result.WriteByte(path[i])
			i++
		}
	}

	return result.String()
}

// SanitizePathForChi converts OpenAPI wildcard paths to Chi-compatible format.
// Chi only allows * at the end of a route, so we convert:
//   - /health/** -> /health/*
//   - /foo/*/bar -> /foo/{wildcard}/bar
//   - /foo/**/bar -> /foo/{wildcard}/bar
func SanitizePathForChi(path string) string {
	// Replace ** with * first
	path = strings.ReplaceAll(path, "**", "*")

	// If path ends with /*, it's valid for chi - leave it
	if strings.HasSuffix(path, "/*") {
		// But check if there are other * in the path
		prefix := path[:len(path)-2]
		if !strings.Contains(prefix, "*") {
			return path
		}
		// Has * in middle, need to fix those
		path = prefix + "/__TRAILING_WILDCARD__"
	}

	// Replace remaining * with {wildcard} (or {wildcard_N} for duplicates)
	count := 0
	for strings.Contains(path, "*") {
		var replacement string
		if count == 0 {
			replacement = "{wildcard}"
		} else {
			replacement = fmt.Sprintf("{wildcard_%d}", count+1)
		}
		path = strings.Replace(path, "*", replacement, 1)
		count++
	}

	// Restore trailing wildcard
	path = strings.ReplaceAll(path, "/__TRAILING_WILDCARD__", "/*")

	return path
}
