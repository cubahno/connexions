
## Fast track

This will start Connexions with a default configuration, sample `petstore` API and contexts.<br/>
Data persisted in the `connexions` volume.

```bash 
docker run -it --rm \
  -p 2200:2200 \
  -v connexions:/app/resources/data \
  cubahno/connexions api
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
                        └───get
        └───service-2
            └───patch
                └───users
                    └───index.json
     
```

!!! note "Directory pattern"

    For `non-openapi` services, the pattern is: <br/>
    `<service-name>/<method>/<path>/<file-name-with-extension>` or <br/>
    `<service-name>/<method>/<path>/index.json` to serve `/path` resource. <br/>
    
    
    See [File structure](/services/#file-structure) for more details.<br/>



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
    See [Contexts](/contexts) for more details.
