package main

import (
    "encoding/json"
    "log"
    "net/http"

    "github.com/cubahno/connexions_plugin"
)

// PetstoreBefore is a middleware that can modify request before it is sent to the server.
// Set in the service config:
// ```
//  middleware:
//	  beforeHandler:
//	    - PetstoreBefore
// ```
func PetstoreBefore(reqResource *connexions_plugin.RequestedResource) ([]byte, error) {
    return nil, nil
}

// PetstoreAfter is a middleware that modifies the response before it is sent to the client.
// Set in the service config:
// ```
//  middleware:
//    afterHandler:
//	    - PetstoreAfter
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
