### About

Callbacks are Go functions that can be shipped with Docker image, allowing request and response contents to be modified on the fly.

### Signature

**Request callback** function signature:
```go
func PetstoreBefore(resource string, request *http.Request) (*http.Request, error) {
    return request, nil
}
```

**Response callback** function signature:
```go
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
```

`connexions_plugin` package is a small package that provides typing support for the callback functions.<br>
User provided functions are built into go plugin and loaded at runtime.

### Callbacks location

**Functions** should be placed in the `callbacks` directory inside the mapped `data` directory.<br>
Filenames are completely arbitrary and can be named as you wish.<br>
But the names of the functions should be:<br/>
- unique
- start with UpperCase
- specified in the service configuration:

**Other conditions:**<br/>
- Package name should be `main`.
- Go version should exactly match the version used to build the Connexions image.

```yaml
services:
  petstore:
    requestTransformer: PetstoreBefore
    responseTransformer: PetstoreAfter
```

`callbacks` directory can be any `go.mod` package which should compile into a shared go plugin.<br>
