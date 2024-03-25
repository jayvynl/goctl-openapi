package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/jayvynl/goctl-openapi/oas3"
	"github.com/zeromicro/go-zero/tools/goctl/plugin"
	"gopkg.in/yaml.v2"
)

var (
	output = flag.String("output", "", `output path of openapi file, default "openapi.json", "-" will output to stdout.`)
	format = flag.String("format", "", `serialization format, "json" or "yaml", default "json".`)
)

func main() {
	log.Default().SetOutput(os.Stderr)
	flag.Parse()

	p, err := plugin.NewPlugin()
	if err != nil {
		log.Fatalf("goctl-openapi: %s\n", err)
	}

	var (
		o = "openapi"
		f = "json"
	)
	if *output != "" {
		o = *output
	}
	if strings.HasSuffix(o, ".json") {
		f = "json"
	} else if strings.HasSuffix(o, ".yml") || strings.HasSuffix(o, ".yaml") {
		f = "yaml"
	} else if *format != "" {
		switch *format {
		case "json":
			f = "json"
		case "yaml", "yml":
			f = "yaml"
		default:
			log.Fatal("format is json or yaml")
		}
		if o != "-" {
			o = fmt.Sprintf("%s.%s", o, f)
		}
	}

	var w io.Writer
	if o == "-" {
		w = os.Stdout
	} else {
		w, err = os.Create(o)
		if err != nil {
			log.Fatalf("goctl-openapi: %s\n", err)
		}
		defer w.(io.Closer).Close()
	}

	doc := oas3.GetDoc(p)
	if f == "json" {
		encoder := json.NewEncoder(w)
		err = encoder.Encode(doc)
		if err != nil {
			log.Fatalf("goctl-openapi: %s\n", err)
		}
	} else {
		encoder := yaml.NewEncoder(w)
		err = encoder.Encode(doc)
		if err != nil {
			log.Fatalf("goctl-openapi: %s\n", err)
		}
	}
}
