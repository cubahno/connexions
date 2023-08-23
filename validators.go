package connexions

import (
	"net/http"
	"regexp"
	"strings"
)

var PlaceholderRegex = regexp.MustCompile(`\{[^\}]*\}`)

func IsValidHTTPVerb(verb string) bool {
	validVerbs := map[string]bool{
		http.MethodGet:     true,
		http.MethodHead:    true,
		http.MethodPost:    true,
		http.MethodPut:     true,
		http.MethodPatch:   true,
		http.MethodDelete:  true,
		http.MethodConnect: true,
		http.MethodOptions: true,
		http.MethodTrace:   true,
	}

	// Convert the input verb to uppercase for case-insensitive comparison
	verb = strings.ToUpper(verb)

	return validVerbs[verb]
}

// IsValidURLResource checks if the given URL resource pattern is valid:
// placeholders contain only alphanumeric characters, underscores, and hyphens
func IsValidURLResource(urlPattern string) bool {
	// Find all pairs of curly brackets in the URL pattern
	matches := ExtractPlaceholders(urlPattern)

	// Validate each pair of curly brackets
	for _, match := range matches {
		// Extract content within curly brackets
		content := match[1 : len(match)-1]
		if content == "" {
			return false
		}

		// Regular expression to match invalid characters within curly brackets
		invalidContentPattern := `[^a-zA-Z0-9_\-/]`
		contentRe := regexp.MustCompile(invalidContentPattern)

		if contentRe.MatchString(content) {
			return false
		}
	}

	return true
}

func ExtractPlaceholders(urlPattern string) []string {
	return PlaceholderRegex.FindAllString(urlPattern, -1)
}
