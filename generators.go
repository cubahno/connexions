package xs

import (
    "github.com/getkin/kin-openapi/openapi3"
)

type ValueMaker func(namePath []string, schema *openapi3.Schema) any

func GenerateContent(schema *openapi3.Schema, valueMaker ValueMaker, namePath []string) any {
    // fast track with value and correctly resolved type
    if len(namePath) > 0 {
        res := valueMaker(namePath, schema)
        if res != nil { // add correct resolved type check
            return res
        }
    }

    if schema.Type == "object" {
        return generateContentObject(schema, valueMaker, namePath)
    } else if schema.Type == "array" {
        return generateContentArray(schema, valueMaker, namePath)
    }

    // try to resolve anything
    return valueMaker(namePath, schema)
}

func generateContentObject(schema *openapi3.Schema, valueMaker ValueMaker, namePath []string) any {
    if namePath == nil {
        namePath = []string{}
    }
    res := map[string]interface{}{}

    if schema.Properties == nil {
        return res
    }

    for name, prop := range schema.Properties {
        res[name] = GenerateContent(prop.Value, valueMaker, append(namePath, name))
    }

    return res
}

func generateContentArray(schema *openapi3.Schema, valueMaker ValueMaker, namePath []string) any {
    if namePath == nil {
        namePath = []string{}
    }

    minItems := int(schema.MinItems)
    var res []any

    for i := 0; i < minItems; i++ {
        res = append(res, GenerateContent(schema.Items.Value, valueMaker, namePath))
    }

    return res
}
