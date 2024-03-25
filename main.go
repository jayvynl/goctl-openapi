package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/jayvynl/goctl-openapi/oas3"
	"github.com/zeromicro/go-zero/tools/goctl/plugin"
	"gopkg.in/yaml.v2"
)

var (
	output = flag.String("filename", "", `openapi file name, default "openapi.json", "-" will output to stdout.`)
	format = flag.String("format", "", `serialization format, "json" or "yaml", default "json".`)
	pretty = flag.Bool("pretty", false, `pretty print of json.`)
)

func main() {
	flag.Parse()

	p, err := plugin.NewPlugin()
	if err != nil {
		fmt.Printf("goctl-openapi: %s\n", err)
		return
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
	} else {
		if *format != "" {
			switch *format {
			case "json":
				f = "json"
			case "yaml", "yml":
				f = "yaml"
			default:
				fmt.Println("goctl-openapi: format must be json or yaml")
				return
			}
		}
		if o != "-" {
			o = fmt.Sprintf("%s.%s", o, f)
		}
	}

	var w *os.File
	if o == "-" {
		w = os.Stdout
	} else {
		w, err = os.Create(path.Join(p.Dir, o))
		if err != nil {
			fmt.Printf("goctl-openapi: %s\n", err)
			return
		}
		defer w.Close()
	}

	doc := oas3.GetDoc(p)
	if f == "json" {
		encoder := json.NewEncoder(w)
		if *pretty {
			encoder.SetIndent("", "  ")
		}
		err = encoder.Encode(doc)
		if err != nil {
			fmt.Printf("goctl-openapi: %s\n", err)
			return
		}
	} else {
		encoder := yaml.NewEncoder(w)
		defer encoder.Close()
		err = encoder.Encode(doc)
		if err != nil {
			fmt.Printf("goctl-openapi: %s\n", err)
			return
		}
	}
}
