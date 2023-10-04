
## Fast track

This will start Connexions with a default configuration, sample `petstore` API and contexts.<br/>
Data persisted in the `connexions` volume.

```bash 
docker run -it --rm \
  -p 2200:2200 \
  -v connexions:/app/resources/data \
  cubahno/connexions api
``` 

## Build your own image

Build your custom Connexions image with your own data and contexts.<br/>
Check directory [examples/docker-image-builder](https://github.com/cubahno/connexions/tree/master/resources/examples/docker-image-builder) 
for full working example with files.

```Dockefile  title="examples/docker-image-builder/Dockerfile"
--8<-- "docker-image-builder/Dockerfile"
```

```bash
docker build . -t my-connexions
```

Run it:

```bash
docker run -it --rm \
  -p 2200:2200 \
  my-connexions api
```

Login to the container to observe the data and contexts:

```bash
docker run -it --rm my-connexions bash
```



## Serve from directory

Connexions resources file structure:
```text
resources
    └─── data
        └───.openapi
            └───service-1
                └───index.yml
        └───.root
            └───get
                └───index.json
        └───service-2
            └───patch
                └───users
                    └───index.json
     
```

!!! note "Directory pattern"

    For `non-openapi` services, the pattern is: <br/>
    `<service-name>/<method>/<path>/<file-name-with-extension>` or <br/>
    `<service-name>/<method>/<path>/index.json` to serve `/path` resource. <br/>
    
    
    See [File structure](services.md/#file-structure) for more details.<br/>



1. Create a directory for your services
2. Create a file with a name `index.json` in there and some contents
3. Run:

```bash
docker run -it --rm \
  -p 2200:2200 \
  -v /path/to/your/services-data:/app/resources/data \
  cubahno/connexions api
```

4. Open `http://localhost:2200` in your browser to see the served `index.json` contents

!!! note "The example above didn't map contexts directory"

    Contexts are essential part to generate meaningful responses.<br/>
    See [Contexts](contexts.md) for more details.


## Pre-built image with Payment Service Providers

Build your custom Connexions image with your own data and contexts.<br/>
Check directory [examples/docker-image-builder](https://github.com/cubahno/connexions/tree/master/resources/examples/docker-with-psp)
for full working example with files.


Run it:

```bash
docker run -it --rm \
  -p 2200:2200 \
  cubahno/connexions-psp api
```


```Dockefile  title="examples/docker-with-psp/Dockerfile"
--8<-- "docker-with-psp/Dockerfile"
```
