package xs

import (
    "github.com/getkin/kin-openapi/openapi3"
    "golang.org/x/text/cases"
    "golang.org/x/text/language"
    "strings"
)

type Replacer func(ctx *ReplaceContext) any

// NULL is used to force resolve to None
const (
    NULL = "__null__"
    NONAME = "__noname__"
)

type PropertyName struct {
    // Name in camel case
    Normalized string
    // Original name from schema
    Original   string
    // Normalized parts joined with dot
    DottedPath string
}

func HasCorrectSchemaType(ctx *ReplaceContext, value any) bool {
    schema, ok := ctx.Schema.(*openapi3.Schema)
    if !ok {
        // TODO(igor): check how to handle with other content schemas
        return true
    }
    return IsCorrectlyReplacedType(value, schema.Type)
}

func ExtractNames(names []string) PropertyName {
    if len(names) == 0 {
        return PropertyName{
            Normalized: NONAME,
            Original:   NONAME,
            DottedPath: NONAME,
        }
    }
    original := names[len(names)-1]
    normalized := ToCamelCase(original)

    parts := make([]string, 0, len(names)-1)
    for _, name := range names[:len(names)-1] {
        parts = append(parts, ToCamelCase(name))
    }

    return PropertyName{
        Normalized: normalized,
        Original:   original,
        DottedPath: strings.Join(parts, "."),
    }
}

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

func ReplaceInHeaders(ctx *ReplaceContext) any {
    if !ctx.State.IsHeader {
        return nil
    }
    return nil
}

func ReplaceFromPredefined(ctx *ReplaceContext) any {
    userData := ctx.Resource.UserReplacements
    if userData == nil {
        return nil
    }

    if res, ok := userData[ctx.Name.Normalized]; ok {
        return res
    }
    return nil
}

func ReplaceFromSchemaFormat(ctx *ReplaceContext) any {
    schema, ok := ctx.Schema.(*openapi3.Schema)
    if !ok {
        return nil
    }

    switch schema.Format {
    case "date":
        return ctx.Faker.Date().Format("2006-01-02")
    case "date-time":
        return ctx.Faker.Date().Format("2006-01-02T15:04:05.000Z")
    case "email":
        return ctx.Faker.Email()
    case "uuid":
        return ctx.Faker.UUID()
    case "password":
        return ctx.Faker.Password(true, true, true, true, true, 12)
    case "hostname":
        return ctx.Faker.DomainName()
    case "ipv4", "ipv6":
        return ctx.Faker.IPv4Address()
    case "uri", "url":
        return ctx.Faker.URL()
    }
    return nil
}

func ReplaceFromSchemaPrimitive(ctx *ReplaceContext) any {
    schema, ok := ctx.Schema.(*openapi3.Schema)
    if !ok {
        return nil
    }
    faker := ctx.Faker

    switch schema.Type {
    case openapi3.TypeString:
        return faker.Word()
    case openapi3.TypeInteger:
        return faker.Uint32()
    case openapi3.TypeNumber:
        return faker.Uint32()
    case openapi3.TypeBoolean:
        return faker.Bool()
    case openapi3.TypeObject:
        // empty object with no response
        return map[string]any{}
    }
    return nil
}

func ReplaceFromSchemaExample(ctx *ReplaceContext) any {
    schema, ok := ctx.Schema.(*openapi3.Schema)
    if !ok {
        return nil
    }
    return schema.Example
}

func ReplaceFallback(ctx *ReplaceContext) any {
    return ctx.Faker.Word()
}
