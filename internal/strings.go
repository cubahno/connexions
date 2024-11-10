package internal

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	matchFirstCap     = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap       = regexp.MustCompile("([a-z0-9])([A-Z])")
	negativeLookAhead = regexp.MustCompile(`\(\?!.*?\)`)
)

// ToSnakeCase converts a string to snake_case case
func ToSnakeCase(input string) string {
	snake := matchFirstCap.ReplaceAllString(input, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	snake = strings.ReplaceAll(snake, ".", "_")
	snake = strings.ReplaceAll(snake, "__", "_")
	snake = strings.Trim(snake, "_")
	return strings.ToLower(snake)
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

func ReplaceNegativeLookahead(pattern string) string {
	return negativeLookAhead.ReplaceAllString(pattern, "")
}
