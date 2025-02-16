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

func ReplaceRequestURL(resource *connexions_plugin.RequestedResource) ([]byte, error) {
    req := resource.Request
    res := req.Clone(req.Context())

    newURL := req.URL
    newURL.Path = "/bar"
    res.URL = newURL

    resource.Request = res

    return nil, nil
}

func ReplaceResponse(resource *connexions_plugin.RequestedResource) ([]byte, error) {
    return []byte("Hallo, Motto!"), nil
}
