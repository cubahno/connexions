package main

import (
	"fmt"
	"github.com/cubahno/connexions"
	"github.com/cubahno/connexions/config"
	"github.com/cubahno/connexions/openapi"
	"github.com/cubahno/connexions/replacers"
)

func main() {
	cfg := config.NewDefaultConfig("")

	schema := &openapi.Schema{
		Type: "object",
		Properties: map[string]*openapi.Schema{
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
	replacer := replacers.CreateValueReplacer(cfg, replacers.Replacers, contexts)
	res := connexions.GenerateContentFromSchema(schema, replacer, nil)
	fmt.Printf("%+v\n", res)

	// will print either:
	// `mmap[age:30 id:111 name:Jane]`
	// or
	// `mmap[age:30 id:222 name:Jane]`
}
