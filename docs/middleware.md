### About

Middleware are Go functions that can be shipped with Docker image, allowing request and response contents to be modified on the fly.

### Signature

Same signature for both request and response middleware functions:
```go
func (reqResource *connexions_plugin.RequestedResource) ([]byte, error) {
    // your code here
}
```

**before request** middleware:<br/>
If middleware returns an error or any non-nil response, the request will be aborted and the response will be returned to the client.

**Middleware** response function:
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

`connexions_plugin` package is a small package that provides typing support for the middleware functions.<br>
User provided functions should be built as go plugins using same go version as `connexions`.

### Middleware location

**Functions** should be placed in the `plugins` directory inside the mapped `data` directory.<br>
Filenames are completely arbitrary and can be named as you wish.<br>
But the names of the middleware functions should be:<br/>
- unique
- start with UpperCase
- specified in the service configuration:

**Other conditions:**<br/>
- Package name should be `main`.
- Go version should exactly match the version used to build the Connexions image.

```yaml
services:
  petstore:
    middleware:
      beforeHandler:
        - PetstoreBefore
      afterHandler:
        - PetstoreAfter
```
