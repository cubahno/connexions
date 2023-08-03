# API Reference

!!! warning "Coming Soon"
    This section is under development. Documentation for the public Go API will be added here.

## Overview

Connexions exposes a public Go API through the `pkg/` package that allows you to:

- Generate mock responses programmatically
- Parse and work with OpenAPI specifications
- Create custom middleware
- Extend the server functionality

## Packages

| Package | Description | Status |
|---------|-------------|--------|
| `pkg/generator` | Response and request generation | 🚧 Upcoming |
| `pkg/schema` | OpenAPI schema parsing and operations | 🚧 Upcoming |
| `pkg/config` | Configuration types and loading | 🚧 Upcoming |
| `pkg/middleware` | HTTP middleware components | 🚧 Upcoming |
| `pkg/loader` | Service and resource loading | 🚧 Upcoming |

## Quick Example

```go
package main

import (
    "github.com/cubahno/connexions/v2/pkg/generator"
    "github.com/cubahno/connexions/v2/pkg/schema"
)

func main() {
    // Load an OpenAPI spec
    // Parse operations
    // Generate responses
    // ... documentation coming soon
}
```

## Contributing

If you'd like to help document the API, contributions are welcome! See the [GitHub repository](https://github.com/cubahno/connexions).

