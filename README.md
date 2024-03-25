goctl-openapi
===

This project is a plugin for [goctl](https://github.com/zeromicro/go-zero/tree/master/tools/goctl).

### Install

```
go install github.com/jayvynl/goctl-openapi@v1.6
```

The version matches goctl version, you should choose goctl-openapi version matches your goctl major and minor version number.

### Usage

Help messages.

```bash
Usage goctl-openapi:
  -filename string
        openapi file name, default "openapi.json", "-" will output to stdout.
  -format string
        serialization format, "json" or "yaml", default "json".
  -pretty
        pretty print of json.
```

Usage example.

```shell
goctl api plugin -plugin goctl-openapi -api example.api -dir .
```
