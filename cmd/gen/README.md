# Code Generation Tools

This directory contains code generation commands for Connexions.

## Commands

| Command | Description | Documentation |
|---------|-------------|---------------|
| `service` | Generate a service from OpenAPI spec or static files | [docs](../../docs/commands/service.md) |
| `simplify` | Simplify large OpenAPI specs | [docs](../../docs/commands/simplify.md) |
| `discover` | Discover services and generate imports (internal) | - |
| `fakes` | Generate fake function list for docs (internal) | - |

## Quick Start

```bash
# Generate a service from an OpenAPI spec
go run github.com/cubahno/connexions/v2/cmd/gen/service@latest \
  -name petstore \
  https://petstore3.swagger.io/api/v3/openapi.json
```

See the [documentation](https://cubahno.github.io/connexions/) for detailed usage.
