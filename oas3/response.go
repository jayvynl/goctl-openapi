package oas3

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/zeromicro/go-zero/tools/goctl/api/spec"
)

func parseResponse(typ string, types map[string]spec.DefineStruct, responses openapi3.ResponseBodies, schemas openapi3.Schemas) *openapi3.ResponseRef {
	if _, ok := responses[typ]; ok {
		return &openapi3.ResponseRef{
			Ref: fmt.Sprintf("#/components/responses/%s", typ),
		}
	}
	schema, err := getSchema(typ, types, schemas)
	if err != nil {
		fmt.Printf("GetSchema of \"%s\": %s", typ, err)
		return nil
	}

	response := &openapi3.ResponseRef{
		Value: &openapi3.Response{
			Description: &DefaultResponseDesc,
			Content: openapi3.Content{
				"application/json": &openapi3.MediaType{
					Schema: schema,
				},
			},
		},
	}
	// swagger editor will complain that []ResponseType is not a valid component name.
	if typ[0] == '[' {
		return response
	}

	responses[typ] = response
	return &openapi3.ResponseRef{
		Ref: fmt.Sprintf("#/components/responses/%s", typ),
	}
}
