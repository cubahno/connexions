Apart from using **Connexions** as a standalone application, you can also use it as a library without 
running any server instance.<br/>

## Installation

```bash
go get github.com/cubahno/connexions
```

### OpenAPI Parser

Load and parse OpenAPI specification file.<br/>
[kin-openapi](https://github.com/getkin/kin-openapi) used as a parser.<br/>

```go
--8<-- "openapi_parser/main.go"
```

### Schema Replacements

```go
--8<-- "schema_replacements/main.go"
```

### Fake Functions

We use fakes from [jaswdr/faker](https://github.com/jaswdr/faker) library.<br/>
Fake functions are gathered in one map with dotted key names.<br/>
All map values implement `FakeFunc` interface.

```go
--8<-- "fake_functions/main.go"
```

### File Operations

```go
--8<-- "file_operations/main.go"
```


### Running Application Server

```go
--8<-- "app_running/main.go"
```
