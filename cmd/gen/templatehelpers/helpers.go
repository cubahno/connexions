package templatehelpers

import (
	"go/token"
	"strings"
	"text/template"
	"unicode"

	"github.com/doordash-oss/oapi-codegen-dd/v3/pkg/codegen"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// GetFuncMap returns a template.FuncMap with all available template functions.
// It combines codegen.TemplateFunctions with our custom functions.
func GetFuncMap() template.FuncMap {
	// Start with codegen's template functions
	funcMap := make(template.FuncMap)
	for k, v := range codegen.TemplateFunctions {
		funcMap[k] = v
	}

	// Add our custom functions (will override if already present)
	titleCaser := cases.Title(language.English)

	funcMap["backtick"] = func(s string) string {
		return "`" + s + "`"
	}

	funcMap["toTitle"] = func(s string) string {
		if s == "" {
			return s
		}
		runes := []rune(s)
		runes[0] = unicode.ToUpper(runes[0])
		return string(runes)
	}

	funcMap["toCamel"] = func(s string) string {
		if s == "" {
			return s
		}
		runes := []rune(s)
		runes[0] = unicode.ToLower(runes[0])
		result := string(runes)

		// If the result is a Go keyword, prefix with underscore to make it valid
		if token.IsKeyword(result) {
			return "_" + result
		}
		return result
	}

	funcMap["title"] = titleCaser.String
	funcMap["lower"] = strings.ToLower

	return funcMap
}

// GetFuncMapWithCustom returns a template.FuncMap with all available template functions
// plus any additional custom functions provided.
func GetFuncMapWithCustom(customFuncs template.FuncMap) template.FuncMap {
	funcMap := GetFuncMap()

	// Add/override with custom functions
	for k, v := range customFuncs {
		funcMap[k] = v
	}

	return funcMap
}
