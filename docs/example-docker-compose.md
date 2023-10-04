
Usage in `docker-compose` is not much different from the [docker](example-docker.md). <br/>

```yaml title="docker-compose.yml"
version: '3'
services:
  api:
    image: cubahno/connexions
    ports:
      - "8080:2200"

  client:
    image: curlimages/curl:latest
    command: ["tail", "-f", "/dev/null"]
    depends_on:
      - api
```

```bash
docker-compose exec client curl http://api:2200/petstore/pets
```

Map the volumes and directories accordingly.
