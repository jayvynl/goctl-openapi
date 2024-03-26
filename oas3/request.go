package oas3

import (
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/jayvynl/goctl-openapi/constant"
	"github.com/zeromicro/go-zero/tools/goctl/api/spec"
)

type (
	rawParsedRequest struct {
		params openapi3.Parameters
		// only contain json fields, form fields will be added in Parse method only when there is no json field.
		schema *openapi3.Schema
	}
	parsedRequest struct {
		params openapi3.Parameters
		body   *openapi3.RequestBodyRef
	}
	requestParser struct {
		rawCache map[string]rawParsedRequest
		cache    map[string]parsedRequest
	}
)

func newRequestParser() requestParser {
	return requestParser{
		rawCache: make(map[string]rawParsedRequest),
		cache:    make(map[string]parsedRequest),
	}
}

func (rp requestParser) parse(
	typ spec.DefineStruct, // RequestType to parse
	types map[string]spec.DefineStruct, // all defined types from api spec
	requests openapi3.RequestBodies, // request body references
	schemas openapi3.Schemas, // schema references, json field of struct type will read and write this map
) rawParsedRequest {
	if item, ok := rp.rawCache[typ.Name()]; ok {
		return item
	}

	embeddedStructs := make([]rawParsedRequest, 0)
	localParams := make(openapi3.Parameters, 0)
	localBodySchema := &openapi3.Schema{
		Type:       openapi3.TypeObject,
		Properties: make(openapi3.Schemas),
	}
	for _, member := range typ.Members {
		// is member a struct type?
		if mt, ok := member.Type.(spec.DefineStruct); ok {
			// embedded struct, recursive parse
			if member.Name == "" {
				// currently go-zero does not support show members of nested struct over 2 levels(include).
				// we can get original type from type definitions in this case.
				if len(mt.Members) == 0 {
					mt = types[mt.Name()]
				}
				es := rp.parse(mt, types, requests, schemas)
				embeddedStructs = append(embeddedStructs, es)
				continue
			}
		}

		fn := getFieldName(member)
		ms, err := getMemberSchema(member, types, schemas)
		if err != nil {
			fmt.Printf("invalid type of %s.%s\n", typ.Name(), member.Name)
		}

		in := getParameterLocation(member.Tags())
		required, allowEmpty := ParseTags(ms, member.Tags())
		if in == "" {
			localBodySchema.Properties[fn] = ms
			if required {
				localBodySchema.Required = append(localBodySchema.Required, fn)
			}
		} else {
			localParams = append(localParams, &openapi3.ParameterRef{
				Value: &openapi3.Parameter{
					Name:            fn,
					In:              in,
					Description:     ms.Value.Description,
					Required:        required,
					AllowEmptyValue: allowEmpty,
					Deprecated:      ms.Value.Deprecated,
					Schema:          ms,
				},
			})
		}
	}

	params := make(openapi3.Parameters, 0)
	bodySchema := &openapi3.Schema{
		Type:        openapi3.TypeObject,
		Title:       typ.Name(),
		Description: strings.Join(typ.Docs, " "),
		Deprecated:  CheckDeprecated(typ.Docs),
		Properties:  make(openapi3.Schemas),
	}

	var (
		tempParams openapi3.Parameters
		tempSchema *openapi3.Schema
	)
	for i := 0; i <= len(embeddedStructs); i++ {
		if i == len(embeddedStructs) {
			tempParams = localParams
			tempSchema = localBodySchema
		} else {
			tempParams = embeddedStructs[i].params
			tempSchema = embeddedStructs[i].schema
		}

	out:
		for _, p := range tempParams {
			for i, op := range params {
				if p.Value.Name == op.Value.Name {
					params[i] = p
					continue out
				}
			}
			params = append(params, p)
		}

		for n, p := range tempSchema.Properties {
			bodySchema.Properties[n] = p
		}
		bodySchema.Required = MergeRequired(bodySchema.Required, tempSchema.Required)
	}

	rpr := rawParsedRequest{
		params: params,
		schema: bodySchema,
	}
	rp.rawCache[typ.Name()] = rpr
	return rpr
}

func (rp requestParser) Parse(
	typ spec.DefineStruct, // RequestType to parse
	types map[string]spec.DefineStruct, // all defined types from api spec
	requests openapi3.RequestBodies, // request body references
	schemas openapi3.Schemas, // schema references, json field of struct type will read and write this map
) (openapi3.Parameters, *openapi3.RequestBodyRef) {
	if item, ok := rp.cache[typ.Name()]; ok {
		return item.params, item.body
	}

	rpr := rp.parse(typ, types, requests, schemas)

	var (
		params   openapi3.Parameters
		schema   *openapi3.Schema
		formBody bool
	)
	if len(rpr.schema.Properties) == 0 && containFormParam(rpr.params) {
		formBody = true
		params = make(openapi3.Parameters, len(rpr.params))
		schema = &openapi3.Schema{
			Type:        openapi3.TypeObject,
			Title:       rpr.schema.Title,
			Description: rpr.schema.Description,
			Deprecated:  rpr.schema.Deprecated,
			Properties:  make(openapi3.Schemas),
		}
		for i, p := range rpr.params {
			if p.Value.In == openapi3.ParameterInQuery {
				schema.Properties[p.Value.Name] = p.Value.Schema
				// if a param both exists in query and form, any one is not required
				if p.Value.Required {
					value := *p.Value
					value.Required = false
					value.AllowEmptyValue = true
					params[i] = &openapi3.ParameterRef{Value: &value}
					continue
				}
			}
			params[i] = p
		}
	} else {
		params = rpr.params
		if len(rpr.schema.Properties) != 0 {
			schema = rpr.schema
		}
	}

	var bodyRef *openapi3.RequestBodyRef
	if schema != nil {
		mediaType := &openapi3.MediaType{
			Schema: &openapi3.SchemaRef{
				Value: schema,
			},
		}
		body := &openapi3.RequestBodyRef{
			Value: &openapi3.RequestBody{
				Description: schema.Description,
				Required:    !formBody,
			},
		}
		if formBody {
			body.Value.Content = openapi3.Content{
				"multipart/form-data":               mediaType,
				"application/x-www-form-urlencoded": mediaType,
			}
		} else {
			body.Value.Content = openapi3.Content{
				"application/json": mediaType,
			}
		}
		requests[typ.Name()] = body
		bodyRef = &openapi3.RequestBodyRef{
			Ref: fmt.Sprintf("#/components/requestBodies/%s", typ.Name()),
		}
	}
	rp.cache[typ.Name()] = parsedRequest{
		params: params,
		body:   bodyRef,
	}
	return params, bodyRef
}

func getMemberSchema(m spec.Member, types map[string]spec.DefineStruct, schemas openapi3.Schemas) (*openapi3.SchemaRef, error) {
	schema, err := getSchema(m.Type.Name(), types, schemas)
	if err != nil {
		return nil, err
	}

	desc := m.GetComment()
	if desc == "" {
		desc = strings.Join(m.Docs, " ")
	}
	deprecated := CheckDeprecated(m.Docs)

	if desc == "" && !deprecated {
		return schema, nil
	}

	// Member is a struct
	if schema.Value == nil {
		// make a copy, because description or deprecated will be changed.
		originalSchema := *schemas[m.Name]
		schema = &originalSchema
	}
	schema.Value.Description = desc
	schema.Value.Deprecated = deprecated
	return schema, nil
}

func containFormParam(params openapi3.Parameters) bool {
	for _, p := range params {
		if p.Value.In == openapi3.ParameterInQuery {
			return true
		}
	}
	return false
}

func getFieldName(m spec.Member) string {
	for _, tag := range m.Tags() {
		if tag.Key == constant.TagKeyJson || tag.Key == constant.TagKeyForm ||
			tag.Key == constant.TagKeyHeader || tag.Key == constant.TagKeyPath {
			if tag.Name == "-" {
				return m.Name
			}
			return tag.Name
		}
	}
	return m.Name
}

func getParameterLocation(tags []*spec.Tag) string {
	for _, tag := range tags {
		switch tag.Key {
		case constant.TagKeyForm:
			return openapi3.ParameterInQuery
		case constant.TagKeyHeader:
			return openapi3.ParameterInHeader
		case constant.TagKeyPath:
			return openapi3.ParameterInPath
		}
	}
	return ""
}
