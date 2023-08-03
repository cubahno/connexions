## Docker Compose

Mount your OpenAPI specs and static files using Docker Compose:

```yaml title="docker-compose.yml"
services:
  connexions:
    image: cubahno/connexions:latest
    ports:
      - "2200:2200"
    volumes:
      # Mount OpenAPI specs - each file becomes a service
      - ./openapi:/app/resources/data/openapi:ro
      # Mount static files - each directory becomes a service
      - ./static:/app/resources/data/static:ro
    environment:
      - LOG_LEVEL=info
      - LOG_FORMAT=text
```

```bash
docker-compose up
```

Test your services:

```bash
curl http://localhost:2200/petstore/pets
```

See the [mounted-services example](https://github.com/cubahno/connexions/tree/master/examples/docker/mounted-services) for a complete working example.
