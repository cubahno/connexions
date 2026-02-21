# App Configuration

App-level configuration for Connexions server.

## File Location

`resources/data/app.yml`

For Docker, mount your custom config:
```yaml
volumes:
  - ./app.yml:/app/resources/data/app.yml:ro
```

## Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `title` | string | `Connexions` | App title displayed in UI |
| `port` | int | `2200` | Server port |
| `homeURL` | string | `/.ui` | URL for UI home page |
| `serviceURL` | string | `/.services` | URL for service endpoints in UI |
| `contextAreaPrefix` | string | `in-` | Prefix for context area replacements |
| `disableUI` | bool | `false` | Disable the web UI |
| `historyDuration` | duration | `5m` | How long to keep request history in memory |
| `editor.theme` | string | `chrome` | Code editor theme in UI |
| `editor.fontSize` | int | `16` | Code editor font size |

## Storage Configuration

Configure shared storage for distributed features (e.g., circuit breaker state sharing across instances).

```yaml
storage:
  type: redis
  redis:
    address: localhost:6379
    password: ""
    db: 0
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `storage.type` | string | `memory` | Storage type: `memory` or `redis` |
| `storage.redis.address` | string | - | Redis address (host:port) |
| `storage.redis.password` | string | - | Redis password |
| `storage.redis.db` | int | `0` | Redis database number |

## Environment Variables

Environment variables override file values:

| Variable | Overrides |
|----------|-----------|
| `ROUTER_HISTORY_DURATION` | `historyDuration` |

## Example

```yaml
title: My API Mock Server
port: 8080
disableUI: false
historyDuration: 10m

editor:
  theme: monokai
  fontSize: 14

storage:
  type: redis
  redis:
    address: redis:6379
```

