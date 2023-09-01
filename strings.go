package connexions

import (
	"regexp"
	"strings"
)

var (
	matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap   = regexp.MustCompile("([a-z0-9])([A-Z])")
)

// ToSnakeCase converts a string to snake_case case
func ToSnakeCase(input string) string {
	snake := matchFirstCap.ReplaceAllString(input, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
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
