# Welcome to Connexions Documentation

[![CI](https://github.com/cubahno/connexions/workflows/CI/badge.svg?event=push)](https://github.com/cubahno/connexions/actions/workflows/ci.yml?query=event%3Apush+branch%3Amaster+workflow%3ACI)
[![Endpoint Badge](https://img.shields.io/endpoint?url=https%3A%2F%2Fgist.githubusercontent.com%2Fcubahno%2F4110782af3ec09dd1ebabc3304756f1f%2Fraw%2Fcovbadge.json&labelColor=%23058FF3&color=%2306C53B)](https://github.com/cubahno/connexions/actions/workflows/ci.yml?query=event%3Apush+branch%3Amaster+workflow%3ACI)
[![GoReportCard](https://goreportcard.com/badge/github.com/cubahno/connexions)](https://goreportcard.com/report/github.com/cubahno/connexions)
[![GoDoc](https://godoc.org/github.com/cubahno/connexions?status.svg)](https://godoc.org/github.com/cubahno/connexions)
[![License](https://img.shields.io/github/license/cubahno/connexions)](https://github.com/cubahno/connexions/blob/master/LICENSE)


**Connexions** allows to define single response for any arbitrary path on the fly.<br/>

## Goals
- provide a simple tool to work with API mocks
- combine multiple APIs into one
- generate meaningful responses

## Features
- Using upstream services with circuit breaker
- Randomized response contents, allowing you to redefine the response for any path
- Modify response contents on the fly by providing custom middleware functions
- Mimic error responses and status codes
- Configurable latency in responses

## Simple start

```bash 

docker run -it --rm \
  -p 2200:2200 \
  -v connexions:/app/resources/data \
  cubahno/connexions api

``` 
