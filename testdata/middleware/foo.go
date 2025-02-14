package main

import (
    "encoding/json"

    "github.com/cubahno/connexions/pkg/plugin"
)

func Foo(resource *plugin.RequestedResource) ([]byte, error) {
    s := map[string]any{
        "foo": "bar",
    }
    res, _ := json.Marshal(s)
    return res, nil
}
