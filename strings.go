package xs

import (
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"regexp"
	"strings"
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

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func ToSnakeCase(input string) string {
	snake := matchFirstCap.ReplaceAllString(input, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	snake = strings.ReplaceAll(snake, "__", "_")
	snake = strings.Trim(snake, "_")
	return strings.ToLower(snake)
}
