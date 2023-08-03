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

# Request/response validation
validate:
  request: true   # Validate incoming requests
  response: false # Validate generated responses

# Caching behavior
cache:
  requests: true  # Cache GET request responses

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

## Validation

Control request and response validation:

```yaml
validate:
  request: true   # Validate requests against OpenAPI spec
  response: false # Validate responses (useful for debugging)
```

When validation fails, the server returns a 400 error with details.

## Caching

Cache responses for GET requests:

```yaml
cache:
  requests: true  # Enable response caching
```

Cached responses are returned for identical GET requests, improving performance.

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
  timeout: 30s
  circuit-breaker:
    threshold: 5
    timeout: 60s
```

When configured, requests are proxied to the upstream server with circuit breaker protection.

## Contexts

Wire context files for realistic data generation:

```yaml
contexts:
  - common:           # Use entire common.yml
  - fake: pet         # Use 'pet' section from fake.yml
  - fake: person      # Use 'person' section from fake.yml
  - petstore:         # Use entire petstore.yml
```

See [Contexts](../contexts.md) for details on context files.

