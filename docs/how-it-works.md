# How It Works

This page explains the request flow through Connexions and how different features interact.

## Request Flow

When a request arrives at Connexions, it passes through a middleware chain:

```
Request → Config Override → Latency/Error → Cache Read → Upstream ──────────→ Response
                                                ↓            ↓ (if failed)
                                            (if hit)    Custom MW → Handler → Cache Write
                                                ↓                                  ↓
                                             Response ←────────────────────────────┘
```

### Middleware Chain

1. **Config Override Middleware** - Applies per-request config overrides from `X-Cxs-*` headers
2. **Latency & Error Middleware** - Simulates network latency and injects errors
3. **Cache Read Middleware** - Returns cached response if available (short-circuits)
4. **Upstream Middleware** - Forwards to real backend; returns response if successful (short-circuits)
5. **Custom Middleware** - Your service-specific middleware (compiled services only)
6. **Handler** - Generates mock response from OpenAPI spec
7. **Cache Write Middleware** - Stores response in cache for future requests

## Per-Request Config Overrides

Override service configuration for individual requests using HTTP headers. 
This is useful for testing, debugging, or handling special cases without modifying the config file.

### Supported Headers

| Header | Values | Description |
|--------|--------|-------------|
| `X-Cxs-Cache-Requests` | `true` / `false` | Enable/disable request caching |
| `X-Cxs-Validate-Request` | `true` / `false` | Enable/disable request validation |
| `X-Cxs-Validate-Response` | `true` / `false` | Enable/disable response validation |
| `X-Cxs-Latency` | Duration (e.g., `100ms`, `1s`) | Override latency |
| `X-Cxs-Upstream-Url` | URL or empty string | Override upstream URL (empty disables upstream) |

### Examples

```bash
# Disable caching for this request
curl -H "X-Cxs-Cache-Requests: false" http://localhost:8080/petstore/pets

# Add 500ms latency
curl -H "X-Cxs-Latency: 500ms" http://localhost:8080/petstore/pets

# Disable upstream proxy (force mock response)
curl -H "X-Cxs-Upstream-Url: " http://localhost:8080/petstore/pets

# Enable response validation for debugging
curl -H "X-Cxs-Validate-Response: true" http://localhost:8080/petstore/pets

# Combine multiple overrides
curl -H "X-Cxs-Latency: 200ms" -H "X-Cxs-Cache-Requests: false" http://localhost:8080/petstore/pets
```

### Case Insensitivity

Headers are case-insensitive. These are all equivalent:

- `x-cxs-cache-requests: false`
- `X-Cxs-Cache-Requests: false`
- `X-CXS-CACHE-REQUESTS: false`

## Latency Simulation

Simulate real-world network conditions to test how your application handles delays.

### Fixed Latency

```yaml
# config.yml
latency: 100ms
```

Every request will be delayed by 100ms.

### Percentile-Based Latency

```yaml
# config.yml
latencies:
  p25: 10ms   # 25% of requests: 10ms
  p50: 50ms   # 25% of requests: 50ms  
  p90: 100ms  # 40% of requests: 100ms
  p99: 500ms  # 9% of requests: 500ms
  p100: 1s    # 1% of requests: 1s
```

This creates a realistic latency distribution where most requests are fast, but some experience higher latency.

## Error Injection

Test error handling by injecting HTTP errors at configurable rates.

```yaml
# config.yml
errors:
  p5: 500   # 5% return 500 Internal Server Error
  p10: 400  # 5% return 400 Bad Request (p10 - p5)
  p15: 429  # 5% return 429 Too Many Requests
```

Percentiles are cumulative - `p10: 400` means requests between p5 and p10 (5%) return 400.

**Flow:**

1. Request arrives
2. Random number generated (0-100)
3. If number ≤ 5 → return 500
4. If number ≤ 10 → return 400
5. If number ≤ 15 → return 429
6. Otherwise → proceed to next middleware

## Upstream Proxy

Forward requests to a real backend service with circuit breaker protection.

```yaml
# config.yml
upstream:
  url: https://api.example.com
  fail-on:
    timeout: 5s
    http-status: "5xx"
```

### How It Works

1. Request arrives at Connexions
2. Upstream middleware forwards request to `https://api.example.com`
3. If successful → return upstream response
4. If failed → proceed to mock handler (fallback)

### Circuit Breaker

The circuit breaker protects against cascading failures:

- **Closed** (normal): Requests flow to upstream
- **Open** (tripped): Requests skip upstream, go directly to mock handler
- **Half-Open** (recovery): Some requests test if upstream is healthy

The circuit opens when:
- At least 3 requests have been made
- Failure ratio ≥ 60%

## Response Caching

Cache GET request responses to improve performance and consistency.

```yaml
# config.yml
cache:
  requests: true
```

### How It Works

1. **Cache Read**: Before processing, check if response exists in cache
2. **Cache Write**: After generating response, store in cache

Cached responses are keyed by `METHOD:URL` and cleared periodically (configurable via `historyDuration` in app settings).

### Cache Behavior

| Request | Cache State | Result |
|---------|-------------|--------|
| GET /pets | Empty | Generate response, cache it |
| GET /pets | Has entry | Return cached response |
| POST /pets | Any | Always generate new response |
| GET /pets/1 | Empty | Generate response, cache it |

## Request Validation

Requests can be validated against the OpenAPI specification at run time.

Validation checks:
- Required parameters
- Parameter types and formats
- Request body schema
- Content-Type headers

Invalid requests return `400 Bad Request` with validation details.

## Response Generation

When no cached or upstream response is available, Connexions generates a mock response:

1. **Find matching operation** in OpenAPI spec
2. **Select response** (prefer 200, then 2xx, then first defined)
3. **Generate response body** - values are resolved in this order:
   - Replace from request headers
   - Replace from path parameters
   - Replace from context files
   - Use schema `example` values
   - Generate based on schema `format` (email, uuid, date, etc.)
   - Generate based on schema primitive type (string, integer, etc.)
   - Fallback to default values

### Static Responses

Override generated responses with static files:

```
static/{service}/{method}/{path}/index.json
```

Example: `static/petstore/get/pets/index.json` overrides `GET /petstore/pets`

### x-static-response Extension

Define static responses directly in your OpenAPI spec:

```yaml
paths:
  /pets:
    get:
      responses:
        '200':
          content:
            application/json:
              x-static-response: |
                [{"id": 1, "name": "Fluffy"}]
```

## Combining Features

Features can be combined for realistic testing scenarios:

```yaml
# config.yml
latencies:
  p50: 50ms
  p99: 200ms

errors:
  p5: 500

upstream:
  url: https://api.example.com

cache:
  requests: true
```

This configuration:
1. Adds realistic latency distribution
2. Injects 5% server errors
3. Tries upstream first, falls back to mock
4. Caches successful GET responses

