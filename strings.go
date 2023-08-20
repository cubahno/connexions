package xs

import (
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"regexp"
	"strings"
	"sync"
)

func ToCamelCase(s string) string {
	words := strings.Fields(strings.ReplaceAll(s, "_", " "))
	for i, word := range words {
		if i > 0 {
			caser := cases.Title(language.English)
			words[i] = caser.String(word)
		} else {
			words[i] = strings.ToLower(word)
		}
	}
	return strings.Join(words, "")
}

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

func MightBeRegexPattern(input string) bool {
	specialChars := []string{"\\", ".", "*", "^", "$", "+", "?", "(", "[", "{", "|"}
	for _, char := range specialChars {
		if strings.Contains(input, char) {
			return true
		}
	}
	return false
}

func ValidateStringWithPattern(input string, pattern string) bool {
	compiledRegex, err := getOrCreateCompiledRegex(pattern)
	if err != nil {
		return false
	}

	return compiledRegex.MatchString(input)
}
