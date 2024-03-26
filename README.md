goctl-openapi
===

This project is a plugin for [goctl](https://github.com/zeromicro/go-zero/tree/master/tools/goctl). It't able to generate openapi specification version 3 file from go-ctl api file.


### Features

- generate correct schema for any level of embedded structure type.
- generate correct schema for complicated type definition like `map[string][]map[int][]*Author`.
- parse parameter constraints from [validate](https://github.com/go-playground/validator) tag.


### Install

This plugin's version and goctl's version should have the same major and minor version, it's recommended to install the matching version. If versions doesn't match, it may not work properly.

For example, if you use goctl v1.6.3, then you should install this plugin with:

```shell
go install github.com/jayvynl/goctl-openapi@v1.6.0
```

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
  -version
        show version and exit.
```

Usage example.

```shell
goctl api plugin -plugin goctl-openapi -api example.api -dir example
```

Take the api file from [example](https://github.com/jayvynl/goctl-openapi/blob/main/example/example.api), [the generated openapi file](https://github.com/jayvynl/goctl-openapi/blob/main/example/openapi.json) can be visualized by [swagger editor](https://editor.swagger.io/?url=https://raw.githubusercontent.com/jayvynl/goctl-openapi/main/example/openapi.json).
