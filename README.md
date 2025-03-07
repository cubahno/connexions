<div style="text-align: center; width:450px;">
    <img src="./resources/docs/images/gotham.svg">
</div>

## Connexions

[![CI](https://github.com/cubahno/connexions/workflows/CI/badge.svg?event=push)](https://github.com/cubahno/connexions/actions/workflows/ci.yml?query=event%3Apush+branch%3Amaster+workflow%3ACI)
[![Endpoint Badge](https://img.shields.io/endpoint?url=https%3A%2F%2Fgist.githubusercontent.com%2Fcubahno%2F4110782af3ec09dd1ebabc3304756f1f%2Fraw%2Fcovbadge.json&labelColor=%23058FF3&color=%2306C53B)](https://github.com/cubahno/connexions/actions/workflows/ci.yml?query=event%3Apush+branch%3Amaster+workflow%3ACI)
[![codecov](https://codecov.io/gh/cubahno/connexions/graph/badge.svg?token=XGCEHYUDH0)](https://codecov.io/gh/cubahno/connexions)
[![GoReportCard](https://goreportcard.com/badge/github.com/cubahno/connexions)](https://goreportcard.com/report/github.com/cubahno/connexions)
[![GoDoc](https://godoc.org/github.com/cubahno/connexions?status.svg)](https://godoc.org/github.com/cubahno/connexions)
[![Go Reference](https://pkg.go.dev/badge/github.com/cubahno/connexions.svg)](https://pkg.go.dev/github.com/cubahno/connexions)
[![License](https://img.shields.io/github/license/cubahno/connexions)](https://github.com/cubahno/connexions/blob/master/LICENSE)


**Connexions** is a library originally inspired by [Connexion](https://github.com/spec-first/connexion).<br/>
Connexion allows you to set up a REST API with Swagger documentation and OAuth2 authentication with minimal effort.<br/>

Connexions takes this one step further by allowing you to define **multiple APIs** not limited to only Swagger and(or) OpenAPI.<br/>
You can define single response for any arbitrary path on the fly.<br/>

## Goals
- provide a simple tool to work with API mocks
- combine multiple APIs into one
- generate meaningful responses

## Features
- Using upstream services with circuit breaker
- Randomized response contents, allowing you to redefine the response for any path
- Modify response contents on the fly by providing custom middleware
- Configurable latencies and errors in responses
- In memory cache for generated responses

<div style="text-align: center; width:auto;">
    <img src="./resources/docs/images/schema-generic.png">
</div>

## Simple start

```bash 
docker run -it --rm \
  -p 2200:2200 \
  -v connexions:/app/resources/data \
  cubahno/connexions api

``` 

We also have a JSON Schema that can be used by IDEs/editors with the Language Server Protocol (LSP) to perform intelligent suggestions, i.e.:
```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/cubahno/connexions/refs/heads/master/resources/json-schema.json
app:
  port: 2200
  disableUI: true
# ...
```

Read full documentation at [cubahno.github.io/connexions](https://cubahno.github.io/connexions//).

[OpenAPI Specification](https://editor.swagger.io/?url=https://raw.githubusercontent.com/cubahno/connexions/master/resources/openapi.yml)

License
===================
Copyright (c) 2023-present

Licensed under the [MIT License](https://github.com/cubahno/connexions/blob/master/LICENSE)


===================

[OpenAPI Specification](https://www.openapis.org/)<br/>
[OpenAPI 3.0 Style Values](https://github.com/OAI/OpenAPI-Specification/blob/master/versions/3.0.2.md#style-values)<br/>
[Operation Object](https://github.com/swagger-api/swagger-spec/blob/master/versions/2.0.md#operation-object)<br/>
[YAML format](https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#format)<br/>
