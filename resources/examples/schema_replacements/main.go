package main

import (
    "fmt"
    "github.com/cubahno/connexions"
)

func main() {
    cfg := connexions.NewDefaultConfig("")

    schema := &connexions.Schema{
        Type: "object",
        Properties: map[string]*connexions.Schema{
            "id": {
                Type:   "string",
                Format: "uuid",
            },
            "name": {
                Type: "string",
            },
            "age": {
                Type:    "integer",
                Minimum: 20,
                Maximum: 30,
            },
        },
    }

    contexts := []map[string]any{
        {"person": map[string]any{"name": "Jane", "age": 33}},
        {
            "id":   []string{"111", "222"},
            "name": "Jane",
        },
    }
    replacer := connexions.CreateValueReplacer(cfg, contexts)
    res := connexions.GenerateContentFromSchema(schema, replacer, nil)
    fmt.Printf("%+v\n", res)

    // will print either:
    // `mmap[age:30 id:111 name:Jane]`
    // or
    // `mmap[age:30 id:222 name:Jane]`
}
