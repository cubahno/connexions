
## Fast track

This will start Connexions with a default configuration and a sample `petstore` API.<br/>
Data persisted in the `connexions` volume.

```bash 
docker run -it --rm \
  -p 2200:2200 \
  -v connexions:/app/resources \
  --name connexions \
  cubahno/connexions api
``` 
