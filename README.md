<div style="text-align: center; width:450px;">
    <img src="https://raw.githubusercontent.com/cubahno/connexions/master/resources/docs/images/gotham.svg">
</div>

## Connexions

[![CI](https://github.com/cubahno/connexions/actions/workflows/ci.yml/badge.svg?branch=master)](https://github.com/cubahno/connexions/actions/workflows/ci.yml?query=branch%3Amaster)
[![Endpoint Badge](https://img.shields.io/endpoint?url=https%3A%2F%2Fgist.githubusercontent.com%2Fcubahno%2F4110782af3ec09dd1ebabc3304756f1f%2Fraw%2Fcovbadge.json&labelColor=%23058FF3&color=%2306C53B)](https://github.com/cubahno/connexions/actions/workflows/ci.yml?query=event%3Apush+branch%3Amaster+workflow%3ACI)
[![codecov](https://codecov.io/gh/cubahno/connexions/graph/badge.svg?token=XGCEHYUDH0)](https://codecov.io/gh/cubahno/connexions)
[![GoReportCard](https://goreportcard.com/badge/github.com/cubahno/connexions)](https://goreportcard.com/report/github.com/cubahno/connexions)
[![GoDoc](https://godoc.org/github.com/cubahno/connexions?status.svg)](https://godoc.org/github.com/cubahno/connexions)
[![Go Reference](https://pkg.go.dev/badge/github.com/cubahno/connexions.svg)](https://pkg.go.dev/github.com/cubahno/connexions)
[![License](https://img.shields.io/github/license/cubahno/connexions)](https://github.com/cubahno/connexions/blob/master/LICENSE)


**Connexions** is a mock server generator for OpenAPI specifications.<br/>
It allows you to define **multiple APIs** and generate meaningful mock responses automatically.<br/>
You can also define static responses for any arbitrary path.<br/>

## Goals
- provide a simple tool to work with API mocks
- combine multiple APIs into one server
- generate meaningful responses

## Features
- **Multiple APIs** on one server - each spec becomes a service with its own URL prefix
- **Upstream proxy** with circuit breaker - forward to real backends with fallback to mocks
- **Latency & error simulation** - test how your app handles delays and failures
- **Custom middleware** - modify requests/responses on the fly
- **Response caching** - cache GET responses for consistency
- **Request validation** - validate against OpenAPI spec

## Real-World Validation

Connexions continuously generates and validates data against **2,200+ real-world OpenAPI specifications** from [cubahno/specs](https://github.com/cubahno/specs):

```
Total: 2215 services, 98464 endpoints
✅ Success: 98464  ❌ Fails: 0
```

## Simple start

```bash 
docker run -it --rm \
  -p 2200:2200 \
  -v connexions:/app/resources/data \
  cubahno/connexions api

``` 

Read full documentation at [cubahno.github.io/connexions](https://cubahno.github.io/connexions/).

[OpenAPI Specification](https://editor.swagger.io/?url=https://raw.githubusercontent.com/cubahno/connexions/master/resources/openapi.yml)

License
===================
Copyright (c) 2023-present

Licensed under the [MIT License](https://github.com/cubahno/connexions/blob/master/LICENSE)
