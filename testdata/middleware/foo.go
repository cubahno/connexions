package main

import (
    "encoding/json"

    "github.com/cubahno/connexions_plugin"
)

func Foo(resource *connexions_plugin.RequestedResource) ([]byte, error) {
    s := map[string]any{
        "foo": "bar",
    }
    res, _ := json.Marshal(s)
    return res, nil
}
