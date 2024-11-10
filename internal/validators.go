package internal

import (
    "log"
    "net/http"
	"regexp"
	"strings"
	"sync"
)

var PlaceholderRegex = regexp.MustCompile(`\{[^\}]*\}`)

var (
	compiledRegexCache = make(map[string]*regexp.Regexp)
	cacheMutex         = sync.Mutex{}
)

// IsValidHTTPVerb checks if the given HTTP verb is valid.
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

// ValidateStringWithPattern checks if the input string matches the given pattern.
func ValidateStringWithPattern(input string, pattern string) bool {
	compiledRegex, err := getOrCreateCompiledRegex(pattern)
	if err != nil {
        log.Printf("Error compiling regex pattern: %s\n", err)
		return false
	}

	return compiledRegex.MatchString(input)
}

// getOrCreateCompiledRegex returns a compiled regex from the cache if it exists,
// otherwise it compiles the regex and adds it to the cache.
func getOrCreateCompiledRegex(pattern string) (*regexp.Regexp, error) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	if cachedRegex, found := compiledRegexCache[pattern]; found {
		return cachedRegex, nil
	}

    pattern = ReplaceNegativeLookahead(pattern)
	compiledRegex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	compiledRegexCache[pattern] = compiledRegex
	return compiledRegex, nil
}
