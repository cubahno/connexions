//go:build exclude
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

// PetstoreBefore is a callback that modifies the request before it is sent to the server.
// Should be set in the service config:
// ```
//	requestTransformer: PetstoreBefore
// ```
func PetstoreBefore(resource string, request *http.Request) (*http.Request, error) {
	res := request.Clone(request.Context())

	oldURL := request.URL.String()
	newURL := request.URL
	newURL.Path = strings.TrimSuffix(request.URL.Path, "/pets")
	res.URL = newURL

	log.Printf("transformed petstore url from %s to: %s\n", oldURL, res.URL.String())
	return res, nil
}

// PetstoreAfter is a callback that modifies the response before it is sent to the client.
// Should be set in the service config:
// ```
//	responseTransformer: PetstoreAfter
// ```
func PetstoreAfter(resource string, request *http.Request, response []byte, isUpstreamResponse bool) ([]byte, error) {
	log.Printf("[PetstoreAfter] req path: %s\n", request.URL.String())
	switch request.Method {
	case http.MethodGet:
		switch resource {
		case "/pets":
			pets := []map[string]any{
				{"name": "dog", "id": 1, "tag": "pet"},
				{"name": "cat", "id": 2, "tag": "pet"},
			}
			log.Println("[PetstoreAfter] returning modified pets")
			return json.Marshal(pets)
		}
	}
	return response, nil
}
