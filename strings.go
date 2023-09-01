package connexions

import (
	"regexp"
	"strings"
	"sync"
)

var (
	matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap   = regexp.MustCompile("([a-z0-9])([A-Z])")
)

var (
	compiledRegexCache = make(map[string]*regexp.Regexp)
	cacheMutex         = sync.Mutex{}
)

func ToSnakeCase(input string) string {
	snake := matchFirstCap.ReplaceAllString(input, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	snake = strings.ReplaceAll(snake, "__", "_")
	snake = strings.Trim(snake, "_")
	return strings.ToLower(snake)
}

func getOrCreateCompiledRegex(pattern string) (*regexp.Regexp, error) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	if cachedRegex, found := compiledRegexCache[pattern]; found {
		return cachedRegex, nil
	}

	compiledRegex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	compiledRegexCache[pattern] = compiledRegex
	return compiledRegex, nil
}

func MaybeRegexPattern(input string) bool {
	specialChars := []string{"\\", ".", "*", "^", "$", "+", "?", "(", "[", "{", "|"}
	for _, char := range specialChars {
		if strings.Contains(input, char) {
			return true
		}
	}
	return false
}
