

```bash
docker build . -t connexions-psp
```

Run it:

```bash
docker run -it --rm \
  -p 2200:2200 \
  connexions-psp api
```

Login to the container to observe the data and contexts:

```bash
docker run -it --rm connexions-psp bash
```
