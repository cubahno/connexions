# Global Settings

Global application settings for the Connexions server. These settings apply to the entire server instance.

!!! tip "JSON Schema Support"
    Add `yaml-language-server` to enable IDE autocompletion:
    ```yaml
    # yaml-language-server: $schema=https://raw.githubusercontent.com/cubahno/connexions/refs/heads/master/resources/json-schema.json
    ```

## Configuration File

Create `config.yml` in your resources directory:

```yaml
app:
  port: 2200
  disableUI: false

services:
  petstore:
    latency: 100ms
    validate:
      request: true
```

## App Settings

```yaml
app:
  port: 2200              # Server port
  homeUrl: /.ui           # UI home URL
  serviceUrl: /.services  # Services API URL
  contextUrl: /.contexts  # Contexts API URL
  disableUI: false        # Disable web UI
  contextAreaPrefix: in-  # Prefix for area-specific contexts
  historyDuration: 5m     # Request history retention
  editor:
    theme: chrome         # Editor theme
    fontSize: 12          # Editor font size
  storage:                # Shared storage for distributed features
    type: memory          # "memory" (default) or "redis"
    redis:                # Required when type is "redis"
      address: localhost:6379
      password: ""
      db: 0
```

### Storage Configuration

For distributed deployments (multiple Connexions instances), configure shared storage:

```yaml
app:
  storage:
    type: redis
    redis:
      address: redis.example.com:6379
      password: secret
      db: 0
```

This enables features like distributed circuit breakers to share state across instances.

## Per-Service Settings

Configure individual services in the global config:

```yaml
services:
  petstore:
    latencies:
      p50: 50ms
      p99: 200ms
    errors:
      p5: 500
    contexts:
      - common:
      - fake: pet
    validate:
      request: true
      response: false
    cache:
      requests: true
```

See [Service Configuration](config/service.md) for detailed options.

## Complete Example

??? note "Full configuration example"

    ```yaml
    app:
      port: 2200
      homeUrl: /.ui
      serviceUrl: /.services
      contextUrl: /.contexts
      disableUI: false
      contextAreaPrefix: in-
      historyDuration: 5m
      editor:
        theme: chrome
        fontSize: 12
      storage:
        type: redis
        redis:
          address: localhost:6379
          password: ""
          db: 0

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
        validate:
          request: true
          response: false
        cache:
          requests: true
        upstream:
          url: https://api.petstore.com
          circuit-breaker:
            timeout: 60s
            min-requests: 3
            failure-ratio: 0.6
    ```

