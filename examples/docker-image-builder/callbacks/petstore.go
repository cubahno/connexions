package main

import (
    "encoding/json"
    "log"
    "net/http"

    "github.com/cubahno/connexions_plugin"
)

// PetstoreBefore is a callback that modifies the request before it is sent to the server.
// Set in the service config:
// ```
//	requestTransformer: PetstoreBefore
// ```
func PetstoreBefore(resource string, request *http.Request) (*http.Request, error) {
    return request, nil
}

// PetstoreAfter is a callback that modifies the response before it is sent to the client.
// Set in the service config:
// ```
//	responseTransformer: PetstoreAfter
// ```
func PetstoreAfter(reqResource *connexions_plugin.RequestedResource) ([]byte, error) {
    log.Printf("[PetstoreAfter] req path: %s\n", reqResource.URL.String())
    switch reqResource.Method {
    case http.MethodGet:
        switch reqResource.Resource {
        case "/pets":
            pets := []map[string]any{
                {"name": "dog", "id": 1, "tag": "pet"},
                {"name": "cat", "id": 2, "tag": "pet"},
            }
            log.Println("[PetstoreAfter] returning modified pets")
            return json.Marshal(pets)
        }
    }
    return reqResource.Response.Data, nil
}
