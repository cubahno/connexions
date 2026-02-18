# Code Generation Tools

This directory contains code generation commands for Connexions.

## Commands

| Command | Description | Documentation |
|---------|-------------|---------------|
| `service` | Generate a service from OpenAPI spec or static files | [docs](../../docs/commands/service.md) |
| `simplify` | Simplify large OpenAPI specs | [docs](../../docs/commands/simplify.md) |
| `discover` | Discover services and generate imports | [usage](#discover) |
| `fakes` | Generate fake function list for docs (internal) | - |

## Quick Start

```bash
# Generate a service from an OpenAPI spec
go run github.com/cubahno/connexions/v2/cmd/gen/service@latest \
  -name petstore \
  https://petstore3.swagger.io/api/v3/openapi.json
```

See the [documentation](https://cubahno.github.io/connexions/) for detailed usage.

## Discover

Scans a directory for `register.go` files and generates an imports file for service auto-registration.

```bash
# Scan default directory (resources/data/services)
go run ./cmd/gen/discover

# Scan custom directory
go run ./cmd/gen/discover pkg/services
```

This generates `cmd/server/services_gen.go` with imports for all discovered services.

Nested directories are supported:
```
pkg/
  adyen/
    v70/register.go  → import "module/pkg/adyen/v70"
    v71/register.go  → import "module/pkg/adyen/v71"
  stripe/
    register.go      → import "module/pkg/stripe"
```
