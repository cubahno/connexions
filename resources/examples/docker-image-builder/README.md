Build your custom Connexions image with your own data and contexts.

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
