package oas3

import (
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/zeromicro/go-zero/tools/goctl/api/spec"
	"github.com/zeromicro/go-zero/tools/goctl/plugin"
)

var DefaultResponseDesc = "A successful response."

func GetDoc(p *plugin.Plugin) (*openapi3.T, error) {
	doc := &openapi3.T{
		OpenAPI:      "3.0.3",
		Components:   newComponents(),
		Info:         getInfo(p.Api.Info.Properties),
		Paths:        openapi3.NewPaths(),
		Servers:      getServers(p.Api.Info.Properties),
		ExternalDocs: getExternalDocs(p.Api.Info.Properties),
	}

	doc.Components.SecuritySchemes["jwt"] = &openapi3.SecuritySchemeRef{
		Value: openapi3.NewJWTSecurityScheme(),
	}
	doc.Security = []openapi3.SecurityRequirement{{"jwt": []string{}}}

	types := make(map[string]spec.DefineStruct) // all defined types from api spec
	for _, typ := range p.Api.Types {
		if ds, ok := typ.(spec.DefineStruct); ok {
			types[ds.Name()] = ds
		}
	}
	fillPaths(p, doc, types, doc.Components.RequestBodies, doc.Components.Responses, doc.Components.Schemas)
	return doc, nil
}

func newComponents() *openapi3.Components {
	return &openapi3.Components{
		Schemas:         make(openapi3.Schemas),
		Parameters:      make(openapi3.ParametersMap),
		Headers:         make(openapi3.Headers),
		RequestBodies:   make(openapi3.RequestBodies),
		Responses:       make(openapi3.ResponseBodies),
		SecuritySchemes: make(openapi3.SecuritySchemes),
		Examples:        make(openapi3.Examples),
		Links:           make(openapi3.Links),
		Callbacks:       make(openapi3.Callbacks),
	}
}

func fillPaths(
	p *plugin.Plugin,
	doc *openapi3.T,
	types map[string]spec.DefineStruct, // all defined types from api spec
	requests openapi3.RequestBodies, // request body references
	responses openapi3.ResponseBodies, // response body references
	schemas openapi3.Schemas, // schema references, json field of struct type will read and write this map
) {
	rp := newRequestParser()

	service := p.Api.Service.JoinPrefix()
	for _, group := range service.Groups {
		for _, route := range group.Routes {
			method := strings.ToUpper(route.Method)
			hasBody := method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch

			tags := getTags(route.AtDoc.Properties)
			if len(tags) == 0 {
				if gn := group.GetAnnotation("group"); len(gn) > 0 {
					tags = []string{gn}
				} else {
					tags = []string{service.Name}
				}
			}

			summary := GetProperty(route.AtDoc.Properties, "summary")
			if summary == "" {
				summary = route.AtDoc.Text
			}
			desc := GetProperty(route.AtDoc.Properties, "description")
			if desc == "" {
				desc = strings.Join(route.Docs, " ")
			}

			var (
				params   openapi3.Parameters
				request  *openapi3.RequestBodyRef
				response *openapi3.ResponseRef
			)
			if typ, ok := route.RequestType.(spec.DefineStruct); ok {
				params, request = rp.Parse(typ, types, requests, schemas)
				if !hasBody {
					request = nil
				}
			}

			var responseTypeName string
			if route.ResponseType != nil {
				responseTypeName = route.ResponseType.Name()
			}

			if responseTypeName == "" {
				response = &openapi3.ResponseRef{
					Value: &openapi3.Response{
						Description: &DefaultResponseDesc,
					},
				}
			} else {
				response = parseResponse(responseTypeName, types, responses, schemas)
			}

			var security *openapi3.SecurityRequirements
			if group.Annotation.Properties["jwt"] != "" {
				security = &openapi3.SecurityRequirements{{"jwt": []string{}}}
			}

			var servers *openapi3.Servers
			if ss := getServers(route.AtDoc.Properties); len(ss) > 0 {
				servers = &ss
			}

			doc.AddOperation(
				ConvertPath(route.Path),
				method,
				&openapi3.Operation{
					Tags:         tags,
					Summary:      summary,
					Description:  desc,
					OperationID:  route.Handler,
					Parameters:   params,
					RequestBody:  request,
					Responses:    openapi3.NewResponses(openapi3.WithStatus(http.StatusOK, response)),
					Security:     security,
					Servers:      servers,
					ExternalDocs: getExternalDocs(route.AtDoc.Properties),
				},
			)
		}
	}
}
