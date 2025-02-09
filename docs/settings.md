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
      historyDuration: 5m
    
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

