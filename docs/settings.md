Add `yaml-language-server` to the top of the file to enable JSON Schema support for your configuration settings.
```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/cubahno/connexions/refs/heads/master/resources/json-schema.json
app:
  port: 2200
  disableUI: true
# ...
```

??? note "Complete example"

    ```yaml
    app:
      port: 2200
      homeUrl: /.ui
      serviceUrl: /.services
      contextUrl: /.contexts
      settingsUrl: /.settings
      serveUI: true
      serveSpec: true
      contextAreaPrefix: in-
      schemaProvider: libopenapi
      createFileStructure: true
      editor:
        theme: chrome
        fontSize: 12
    
    services:
      petstore:
        latencies:
          p25: 10ms
          p99: 20ms
          p100: 25ms

        errors:
          p10: 400
          p20: 500

        contexts:
        - common:
        - fake: pet
        - fake: gamer

        parseConfig:
          maxLevels: 6
          maxRecursionLevels: 2

        validate:
          request: true
          response: false

        cache:
          schema: true

    ```

!!! note "Non-configurable values"

    This document provides a complete list of all configurable settings.<br>
    However, there are more values that can be set, but only when using `connexions` as library.<br>


### app

Main application settings.

#### port

Type: `number`<br/>
Default: `2200`<br/>


#### homeUrl
Default: `/.ui`<br/>

#### serviceUrl


#### contextUrl


#### settingsUrl


#### serveUi

Disable UI if not needed by settings the value to `false`. <br/>
Default: `true`<br/>


#### serveSpec
Adds swagger-ui to each service.

Default: `true`<br/>


#### contextAreaPrefix

Set sub-contexts for replacements in path, header or any other supported place
```yaml title="Example"
in-path:
  pet_id: 123
  
in-header:
  x-api-key: abcde
```

See <a href="/contexts/#path-and-headers">Contexts</a> documentation for more details.

Default: `in-`<br/>


#### schemaProvider
There're 2 choices:<br/>
 - **libopenapi**: <a href="https://github.com/pb33f/libopenapi" target="_blank">view</a><br/>
 - **kin-openapi**: <a href="https://github.com/getkin/kin-openapi" target="_blank">view</a>

Both have their pros and cons.<br/>
`libopenapi` provides circular reference support, OpenAPI 3.1 support but not widely adopted yet.<br/>
`kin-openapi` is widely adopted, but doesn't support circular references and OpenAPI 3.1 yet. <br/>
But it provides request / response validation.<br/>

This setting might be removed in the future.<br/>
It is here to provide an easy way to switch between the 2 libraries while picking the right choice.<br/>

Default: `libopenapi`<br/>


#### createFileStructure
Pre-create the file structure for the resources.<br/>
When set to `true`, the file structure will be created on startup:<br/>
```text
resources:
  |- data:
      |- contexts:
      |- services
      config.yml
```

Default: `true`<br/>


#### editor
Configuration for UI request / response editor.

###### theme

Default: `chrome`<br/>
For a list of themes, see <a href="https://github.com/ajaxorg/ace/tree/master/src/theme" target="_blank">Ace Editor list.</a>

###### fontSize
Default: `12`<br/>


### services
Map of service configurations, where the `key` is the service name.<br/>
Service name is the first part of the path.<br/>
e.g. `/petstore/v1/pets -> petstore`<br/>
In case, service name is omitted, `root` name will be used internally in directory structure.

#### latency
Applies `time.Duration` latency to all service responses.

Default: `0s`<br/>

#### latencies
Applies latency to specific percentiles of the service responses.

```yaml title="Example"
p25: 10ms
p99: 20ms
p100: 25ms
```

The keys are the percentile values, and the values are the latencies to be applied.<br/>

#### errors

```yaml title="Example"
p10: 400
p20: 500
```

The keys are the percentile values, and the values are the status codes to be returned.<br/>

#### contexts

The name of the context to use when substituting the values in the request/response.<br/>
Applied in the order of definition.<br/>

See <a href="/contexts">Contexts</a> documentation for more details on working with contexts.

Default: 
```yaml
contexts:
  - common:
  - fake:
```


#### parseConfig    

Optimize openapi schema parsing for better performance

###### maxLevels
Parse only the first `N` levels of the schema<br/>

Default: `6`<br/>

###### maxRecursionLevels

if schema has more than `N` levels of recursion, stop parsing<br/>

Default: `2`<br/>


#### validate

Validate incoming requests and outgoing responses to conform to the schema.<br/>
Only works with `kin-openapi` provider now.<br/>

###### request

Default: `false`<br/>


###### response

Default: `true`<br/>


#### cache

Various caching options.

###### schema
Cache the parsed schema in memory for faster access.

Default: `true`<br/>
