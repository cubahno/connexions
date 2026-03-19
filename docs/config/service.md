# Service Configuration

Each service can have its own `config.yml` file that controls runtime behavior like latency simulation, error injection, validation, and caching.

## Location

For compiled Go services, place `config.yml` in the `setup/` directory:

```
services/petstore/
└── setup/
    ├── config.yml      # Service configuration
    ├── codegen.yml     # Code generation settings
    └── openapi.yml     # OpenAPI specification
```

For Docker with mounted specs, use a directory structure with `config.yml` alongside your spec:

```
openapi/
└── petstore/
    ├── openapi.yml     # OpenAPI spec (must be named openapi.yml/yaml/json)
    └── config.yml      # Service config
```

Note: Flat specs like `openapi/petstore.yml` don't support config files - use the directory structure above.

## Configuration Options

### Service Name

```yaml
name: petstore
```

The `name` property defines the **URL prefix** for all routes in this service.

For example, if your OpenAPI spec has `/pets` and `/pets/{id}`, they become:
- `GET /petstore/pets`
- `GET /petstore/pets/{id}`

This allows multiple APIs to coexist on the same server without route conflicts.

**How the name is determined:**

1. If `name` is set in `config.yml` → uses that name
2. Otherwise → inferred from directory name (e.g., `openapi/stripe/` → `stripe`)
3. For flat specs → inferred from filename (e.g., `openapi/petstore.yml` → `petstore`)

### Latency

```yaml
# Simulated latency for responses
latency: 100ms

# Latency distribution by percentile
latencies:
  p50: 50ms
  p90: 100ms
  p99: 500ms

# Error injection by percentile
errors:
  p5: 500   # 5% of requests return 500
  p10: 400  # 5% return 400 (p10 - p5)

# Caching behavior
cache:
  requests: true  # Cache GET request responses
  replay:
    ttl: 24h
    auto-replay: false
    upstream-only: false
    endpoints:
      /path/{id}:
        POST:
          match:
            - data.name

# OpenAPI spec simplification
spec:
  simplify: false
  optional-properties:
    number: 5     # Keep max 5 optional properties
```

## Latency Simulation

Simulate real-world network conditions:

```yaml
# Fixed latency
latency: 100ms

# Or percentile-based distribution
latencies:
  p25: 10ms   # 25% of requests: 10ms
  p50: 50ms   # 25% of requests: 50ms  
  p90: 100ms  # 40% of requests: 100ms
  p99: 500ms  # 9% of requests: 500ms
  p100: 1s    # 1% of requests: 1s
```

## Error Injection

Test error handling by injecting HTTP errors:

```yaml
errors:
  p5: 500   # 5% return 500 Internal Server Error
  p10: 400  # 5% return 400 Bad Request
  p15: 429  # 5% return 429 Too Many Requests
```

Percentiles are cumulative - `p10: 400` means requests between p5 and p10 (5%) return 400.

## Caching

Cache responses for GET requests:

```yaml
cache:
  requests: true  # Enable response caching
```

Cached responses are returned for identical GET requests, improving performance.

## Replay

Record API responses and replay them on subsequent requests that match specific request body fields. Works like VCR — record once, replay on match.

```yaml
cache:
  replay:
    ttl: 24h              # How long recordings are kept (default: 24h)
    upstream-only: false   # Only record upstream responses (default: false)
    auto-replay: false     # Activate without header (default: false)
    endpoints:
      /foo/{f-id}/bar/{b-id}:
        POST:
          match:
            - data.name
            - data.address.zip
```

### How It Works

1. A request comes in with the `X-Cxs-Replay` header (or to an `auto-replay` endpoint)
2. Specified fields are extracted from the request body using dotted paths
3. A content-addressed key is built from the method, path pattern, and extracted values
4. If a recording exists for that key → return it immediately with `X-Cxs-Source: replay`
5. If no recording exists → forward to downstream, capture the response, store it, and return it

### Activation

Replay activates in two ways:

**Header-based (default):** Send the `X-Cxs-Replay` header to activate replay for any request.

```bash
# Empty header — uses match fields from config
curl -X POST /svc/foo/123/bar/456 \
  -H "X-Cxs-Replay:" \
  -d '{"data": {"name": "Jane", "address": {"zip": "12345"}}}'

# Header with fields — overrides config match fields
curl -X POST /svc/foo/123/bar/456 \
  -H "X-Cxs-Replay: data.name,data.address.zip" \
  -d '{"data": {"name": "Jane", "address": {"zip": "12345"}}}'
```

**Auto-replay:** Set `auto-replay: true` in config to activate for configured endpoints without requiring the header.

```yaml
cache:
  replay:
    auto-replay: true
    endpoints:
      /users:
        POST:
          match:
            - email
```

### Match Fields

Match fields are dotted paths into the JSON request body. Supported formats:

| Format | Example | Description |
|--------|---------|-------------|
| Simple | `data.name` | Traverse nested objects |
| Array index | `data.items[0].name` | Access specific array element |
| Array wildcard | `data.items.name` | Search each array element, return first match |

### Key Design

The replay key is a SHA-256 hash of: `METHOD:pattern_path|field1=value1|field2=value2|...`

- **Pattern path** is used (e.g., `/foo/{id}/bar`), not the actual URL — so different path parameter values share recordings
- **Fields are sorted** alphabetically for determinism
- Only the matched body fields matter — other body content is ignored

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `ttl` | duration | `24h` | How long recordings are kept |
| `upstream-only` | bool | `false` | Only record responses from upstream services |
| `auto-replay` | bool | `false` | Activate for configured endpoints without requiring the header |
| `endpoints` | map | — | Path patterns → methods → match fields |

### Upstream-Only Mode

When `upstream-only: true`, only responses from upstream services are recorded. If the response is not from upstream (e.g., generated or cached), the middleware returns a `502 Bad Gateway` error instead of passing the response through. This makes it explicit to the caller that the recording was skipped.

```yaml
cache:
  replay:
    upstream-only: true
    endpoints:
      /external-api/search:
        POST:
          match:
            - query
            - filters.category
```

This is useful when you want to capture real upstream responses for later replay and need a clear signal when the upstream is not available.

### Example: Full Configuration

```yaml
name: my-service
upstream:
  url: https://api.example.com
cache:
  requests: true
  replay:
    ttl: 12h
    upstream-only: true
    auto-replay: false
    endpoints:
      /search:
        POST:
          match:
            - query
            - filters.category
      /users/{user-id}/orders:
        POST:
          match:
            - items[0].product_id
            - shipping.method
        PUT:
          match:
            - order_id
```

## Spec Simplification

Reduce complexity of large OpenAPI specs:

```yaml
spec:
  simplify: false  # Enable/disable simplification
  optional-properties:
    number: 5      # Keep max N optional properties per schema
    # range: "1-6" # Or random range
```

This helps with specs that have schemas with many optional fields.

## Upstream Proxy

Forward requests to a real backend:

```yaml
upstream:
  url: https://api.example.com
  timeout: 5s               # Request timeout (default: 5s)
  headers:
    X-Custom-Header: value
  fail-on:                   # Return these statuses directly (default: 400)
    - exact: 400
  circuit-breaker:
    trip-on-status:          # Only these statuses count as CB failures
      - range: "500-599"
```

When configured, requests are proxied to the upstream server.
If the upstream fails (timeout, network error, or error status), Connexions falls back to generating mock responses.

### Fail-On

Control which upstream error status codes are returned directly to the client instead of falling back to the generator:

```yaml
upstream:
  url: https://api.example.com
  fail-on:                     # Return these statuses directly (no generator fallback)
    - exact: 400
    - range: "401-403"
```

| Configuration | Behavior |
|---|---|
| Not set (omitted) | Default: only `400` is returned directly |
| `fail-on: []` | Disabled — all errors fall back to the generator |
| `fail-on: [{range: "400-499"}]` | All 4xx returned directly |

`fail-on` and `trip-on-status` are independent — `fail-on` controls what the client sees,
`trip-on-status` controls what counts toward circuit breaker failures.

### Circuit Breaker

Protect against cascading failures with circuit breaker:

```yaml
upstream:
  url: https://api.example.com
  circuit-breaker:
    timeout: 60s        # Time in open state before half-open (default: 60s)
    max-requests: 1     # Max requests in half-open state (default: 1)
    interval: 30s       # Interval to clear counts in closed state (default: 0)
    min-requests: 3     # Min requests before tripping (default: 3)
    failure-ratio: 0.6  # Failure ratio to trip (default: 0.6)
```

**Circuit breaker states:**

- **Closed**: Normal operation, requests pass through
- **Open**: After failure threshold, requests are blocked
- **Half-Open**: After timeout, limited requests test if service recovered

When app-level storage is configured with `type: redis`, circuit breaker state is automatically shared across instances.

## Response Headers

Connexions adds the following headers to responses:

| Header | Values | Description |
|--------|--------|-------------|
| `X-Cxs-Source` | `upstream`, `cache`, `generated`, `replay` | Indicates where the response came from |
| `X-Cxs-Duration` | e.g. `1.234ms` | Request processing time |

## Contexts

See [Contexts](../contexts.md) for details on context files.

