package xs

import (
    "github.com/getkin/kin-openapi/openapi3"
)

type ValueMaker func(schema *openapi3.Schema, state *generatorState) any

type generatorState struct {
    NamePath []string
    Example  any
    IsHeader bool
}

func (s *generatorState) addPath(name string) *generatorState {
    namePath := s.NamePath
    if len(namePath) == 0 {
        namePath = []string{}
    }

    return &generatorState{
        NamePath: append(namePath, name),
        Example:  s.Example,
        IsHeader: s.IsHeader,
    }
}

func (s *generatorState) header() *generatorState {
    return &generatorState{
        NamePath: s.NamePath,
        Example:  s.Example,
        IsHeader: true,
    }
}

func GenerateHeaders(headers openapi3.Headers, valueMaker ValueMaker, state *generatorState) any {
    if state == nil {
        state = &generatorState{}
    }

    res := map[string]interface{}{}

    for name, headerRef := range headers {
        header := headerRef.Value
        params := header.Parameter
        res[name] = GenerateContent(params.Schema.Value, valueMaker, state.addPath(name))
    }
    return res
}

func GenerateContent(schema *openapi3.Schema, valueMaker ValueMaker, state *generatorState) any {
    if state == nil {
        state = &generatorState{}
    }
    // fast track with value and correctly resolved type
    if len(state.NamePath) > 0 {
        if res := valueMaker(schema, state); res != nil { // add correct resolved type check
            return res
        }
    }

    if schema.Type == "object" {
        return generateContentObject(schema, valueMaker, state)
    } else if schema.Type == "array" {
        return generateContentArray(schema, valueMaker, state)
    }

    // try to resolve anything
    return valueMaker(schema, state)
}

func generateContentObject(schema *openapi3.Schema, valueMaker ValueMaker, state *generatorState) any {
    if state == nil {
        state = &generatorState{}
    }
    res := map[string]interface{}{}

    if schema.Properties == nil {
        return res
    }

    for name, prop := range schema.Properties {
        res[name] = GenerateContent(prop.Value, valueMaker, state.addPath(name))
    }

    return res
}

func generateContentArray(schema *openapi3.Schema, valueMaker ValueMaker, state *generatorState) any {
    if state == nil {
        state = &generatorState{}
    }
    minItems := int(schema.MinItems)
    var res []any

    for i := 0; i < minItems; i++ {
        res = append(res, GenerateContent(schema.Items.Value, valueMaker, state))
    }

    return res
}
