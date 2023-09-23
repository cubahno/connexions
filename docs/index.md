# Welcome to Connexions Documentation

[![CI](https://github.com/cubahno/connexions/workflows/CI/badge.svg?event=push)](https://github.com/cubahno/connexions/actions/workflows/ci.yml?query=event%3Apush+branch%3Amaster+workflow%3ACI)
[![Cov](https://img.shields.io/endpoint?url=https://gist.githubusercontent.com/cubahno/4110782af3ec09dd1ebabc3304756f1f/raw/covbadge.json)](https://github.com/cubahno/connexions/actions/workflows/ci.yml?query=event%3Apush+branch%3Amaster+workflow%3ACI)


**Connexions** is a library inspired by [Connexion](https://github.com/spec-first/connexion).<br/>
Connexion allows you to set up a REST API with Swagger documentation and OAuth2 authentication with minimal effort.<br/>

**Connexions** takes this one step further by allowing you to define **multiple APIs** not limited to only Swagger and(or) OpenAPI.<br/>
You can define single response for any arbitrary path on the fly.<br/>

## Goal:
- simplify the development process
- combine multiple APIs into one
- generate meaningful responses

## Features:
- Randomized response contents, allowing you to redefine the response for any path in a locale of your choice
- Mimic error responses and status codes
- Configurable latency in responses

## Simple start:

```bash 

docker run -it --rm \
  -p 2200:2200 \
  -v connexions:/app/resources \
  cubahno/connexions api

``` 
