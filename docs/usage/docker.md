## Quick Start

Mount your OpenAPI specs and get a mock server instantly:

```bash
docker run -p 2200:2200 \
  -v ~/my-specs:/app/resources/data/openapi \
  cubahno/connexions:latest
```

## Mount OpenAPI Specs

Place your OpenAPI specs (`.yml`, `.yaml`, `.json`) in a directory and mount it:

```bash
docker run -p 2200:2200 \
  -v /path/to/specs:/app/resources/data/openapi \
  cubahno/connexions:latest
```

Each spec file becomes a service. For example, `petstore.yml` creates endpoints at `/petstore/...`.

## Mount Static Files

For static response files, organize them by service name and HTTP method:

```text
my-static/
└── myapi/
    └── users/
        ├── GET.json      # GET /myapi/users
        └── {id}/
            └── GET.json  # GET /myapi/users/{id}
```

Mount the directory:

```bash
docker run -p 2200:2200 \
  -v /path/to/static:/app/resources/data/static \
  cubahno/connexions:latest
```

## Mount Both

You can mount both OpenAPI specs and static files:

```bash
docker run -p 2200:2200 \
  -v ~/my-specs:/app/resources/data/openapi \
  -v ~/my-static:/app/resources/data/static \
  cubahno/connexions:latest
```

## Hot Reload

The server watches for file changes. When you modify a spec or static file:

1. File watcher detects the change
2. Service is regenerated
3. Server restarts automatically

No need to restart the container manually.

## Directory Structure

```text
/app/resources/data/
├── openapi/     # Mount OpenAPI specs here (auto-generates services)
├── static/      # Mount static files here (auto-generates services)
└── services/    # Generated Go services (managed by connexions)
```

## Examples

See the [mounted-services example](https://github.com/cubahno/connexions/tree/master/examples/docker/mounted-services) for a complete working example.

## Pre-generated Services

If you need custom middleware or want to pre-generate Go services, see the [pre-generated-services example](https://github.com/cubahno/connexions/tree/master/examples/docker/pre-generated-services).
