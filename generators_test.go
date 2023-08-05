package xs

import (
    "encoding/json"
    "github.com/getkin/kin-openapi/openapi3"
    "github.com/stretchr/testify/assert"
    "testing"
)

func TestGenerateContentObject(t *testing.T) {
    t.Run("test case 1", func(t *testing.T) {
        schema := &openapi3.Schema{}
        src := `
        {
            "type":"object",
            "properties": {
                "name": {
                    "type": "object",
                    "properties": {
                        "first": {
                            "type": "string"
                        },
                        "last": {
                            "type": "string"
                        }
                    }
                },
                "age": {
                    "type": "integer"
                }
            }
        }`

        err := json.Unmarshal([]byte(src), schema)
        if err != nil {
            t.Fail()
        }

        valueMaker := func(schema *openapi3.Schema, state *GeneratorState) any {
            namePath := state.NamePath
            for _, name := range namePath {
                if name == "first" {
                    return "Jane"
                } else if name == "last" {
                    return "Doe"
                } else if name == "age" {
                    return 21
                }
            }
            return nil
        }
        res := generateContentObject(schema, valueMaker, nil)

        expected := `{"age":21,"name":{"first":"Jane","last":"Doe"}}`
        resJs, _ := json.Marshal(res)
        assert.Equal(t, expected, string(resJs))
    })
}
