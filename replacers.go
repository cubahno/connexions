package xs

import (
    "github.com/getkin/kin-openapi/openapi3"
)

type Replacer func(ctx *ReplaceContext) any

// NULL is used to force resolve to None
const (
    NULL = "__null__"
    NONAME = "__noname__"
)

func HasCorrectSchemaType(ctx *ReplaceContext, value any) bool {
    schema, ok := ctx.Schema.(*openapi3.Schema)
    if !ok {
        // TODO(igor): check how to handle with other content schemas
        return true
    }
    return IsCorrectlyReplacedType(value, schema.Type)
}

func ExtractNames(names []string) (string, string) {
    if len(names) == 0 {
        return NONAME, NONAME
    }
    original := names[len(names)-1]
    normalized := original

    return normalized, original
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

    if res, ok := userData[ctx.Name]; ok {
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
