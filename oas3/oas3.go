package oas3

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/jayvynl/goctl-openapi/constant"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/tools/goctl/api/spec"
	"github.com/zeromicro/go-zero/tools/goctl/plugin"
)

func GetDoc(p *plugin.Plugin) *openapi3.T {
	doc := &openapi3.T{
		OpenAPI:      "3.0",
		Components:   NewComponents(),
		Info:         GetInfo(p.Api.Info.Properties),
		Paths:        openapi3.NewPaths(),
		Servers:      GetServers(p.Api.Info.Properties),
		ExternalDocs: GetExternalDocs(p.Api.Info.Properties),
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
	FillPaths(p, doc, types, doc.Components.RequestBodies, doc.Components.Responses, doc.Components.Schemas)
	return doc
}

func NewComponents() *openapi3.Components {
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

func GetInfo(properties map[string]string) *openapi3.Info {
	title, _ := GetProperty(properties, constant.ApiInfoTitle)
	version, _ := GetProperty(properties, constant.ApiInfoVersion)
	desc, _ := GetProperty(properties, constant.ApiInfoDesc)
	author, _ := GetProperty(properties, constant.ApiInfoAuthor)
	email, _ := GetProperty(properties, constant.ApiInfoEmail)
	info := &openapi3.Info{
		Title:       title,
		Description: desc,
		Version:     version,
	}
	if author != "" || email != "" {
		info.Contact = &openapi3.Contact{
			Name:  author,
			Email: email,
		}
	}
	return info
}

func GetServers(properties map[string]string) openapi3.Servers {
	urls, _ := GetProperty(properties, constant.ApiInfoServers)
	urls = strings.TrimSpace(urls)
	if urls == "" {
		return nil
	}

	urlList := strings.Split(urls, ",")
	servers := make(openapi3.Servers, len(urlList))
	for i, url := range urlList {
		servers[i] = &openapi3.Server{URL: url}
	}
	return servers
}

func GetExternalDocs(properties map[string]string) *openapi3.ExternalDocs {
	url, _ := GetProperty(properties, constant.ApiInfoExternalDocs)
	url = strings.TrimSpace(url)
	if url == "" {
		return nil
	}
	return &openapi3.ExternalDocs{
		URL: url,
	}
}

func GetTags(properties map[string]string) []string {
	names, _ := GetProperty(properties, constant.ApiInfoTags)
	names = strings.TrimSpace(names)
	if names == "" {
		return nil
	}

	return strings.Split(names, ",")
}

func FillPaths(
	p *plugin.Plugin,
	doc *openapi3.T,
	types map[string]spec.DefineStruct, // all defined types from api spec
	requests openapi3.RequestBodies, // request body references
	responses openapi3.ResponseBodies, // response body references
	schemas openapi3.Schemas, // schema references, json field of struct type will read and write this map
) {
	rp := NewRequestParser()

	service := p.Api.Service.JoinPrefix()
	for _, group := range service.Groups {
		for _, route := range group.Routes {
			method := strings.ToUpper(route.Method)
			hasBody := method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch

			tags := GetTags(route.AtDoc.Properties)
			if len(tags) == 0 {
				if gn := group.GetAnnotation("group"); len(gn) > 0 {
					tags = []string{service.Name, gn}
				} else {
					tags = []string{service.Name}
				}
			}

			summary, _ := GetProperty(route.AtDoc.Properties, "summary")
			if summary == "" {
				summary = route.AtDoc.Text
			}
			desc, _ := GetProperty(route.AtDoc.Properties, "description")
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
			if typ := route.ResponseType.Name(); typ != "" {
				response = ParseResponse(typ, types, responses, schemas)
			}

			var security *openapi3.SecurityRequirements
			if group.Annotation.Properties["jwt"] != "" {
				security = &openapi3.SecurityRequirements{{"jwt": []string{}}}
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
					ExternalDocs: GetExternalDocs(route.AtDoc.Properties),
				},
			)
		}
	}
}

func ParseResponse(typ string, types map[string]spec.DefineStruct, responses openapi3.ResponseBodies, schemas openapi3.Schemas) *openapi3.ResponseRef {
	if _, ok := responses[typ]; ok {
		return &openapi3.ResponseRef{
			Ref: fmt.Sprintf("#/components/responseBodies/%s", typ),
		}
	}
	schema, err := GetSchema(typ, types, schemas)
	if err != nil {
		log.Printf("GetSchema of \"%s\": %s", typ, err)
		return nil
	}

	desc := "A successful response."
	responses[typ] = &openapi3.ResponseRef{
		Value: &openapi3.Response{
			Description: &desc,
			Content: openapi3.Content{
				"application/json": &openapi3.MediaType{
					Schema: schema,
				},
			},
		},
	}
	return &openapi3.ResponseRef{
		Ref: fmt.Sprintf("#/components/responseBodies/%s", typ),
	}
}

func GetParameterLocation(tags []*spec.Tag) string {
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

func GetStructSchema(typ spec.DefineStruct, types map[string]spec.DefineStruct, schemas openapi3.Schemas) *openapi3.SchemaRef {
	if _, ok := schemas[typ.Name()]; ok {
		return &openapi3.SchemaRef{Ref: fmt.Sprintf("#/components/schemas/%s", typ.Name())}
	}

	allOf := make(openapi3.SchemaRefs, 0)
	schema := &openapi3.SchemaRef{
		Value: &openapi3.Schema{
			Type:        openapi3.TypeObject,
			Title:       typ.Name(),
			Description: strings.Join([]string(typ.Docs), " "),
			Deprecated:  CheckDeprecated(typ.Docs),
			Properties:  make(openapi3.Schemas),
		},
	}
	for _, m := range typ.Members {
		fn := GetFieldName(m)
		// is member a struct type?
		if mt, ok := m.Type.(spec.DefineStruct); ok {
			// embedded struct, recursive parse
			// currently go-zero does not support show members of nested struct over 2 levels(include).
			// we can get original type from type definitions in this case.
			mt = types[mt.Name()]
			ms := GetStructSchema(mt, types, schemas)
			if m.Name == "" {
				allOf = append(allOf, ms)
				continue
			} else {
				schema.Value.Properties[fn] = ms
			}
		} else {
			memberSchema, err := GetMemberSchema(m, types, schemas)
			if err != nil {
				log.Printf("invalid type of %s.%s\n", typ.Name(), m.Name)
				continue
			}
			schema.Value.Properties[fn] = memberSchema
		}

		if ParseTags(schema.Value.Properties[fn], m.Tags()) {
			schema.Value.Required = append(schema.Value.Required, fn)
		}
	}

	if len(allOf) != 0 {
		allOf = append(allOf, schema)
		schema = &openapi3.SchemaRef{
			Value: &openapi3.Schema{
				Title:       schema.Value.Title,
				Description: schema.Value.Description,
				Deprecated:  schema.Value.Deprecated,
				AllOf:       allOf,
			},
		}
	}
	schemas[typ.Name()] = schema
	return &openapi3.SchemaRef{Ref: fmt.Sprintf("#/components/schemas/%s", typ.Name())}
}

func GetMemberSchema(m spec.Member, types map[string]spec.DefineStruct, schemas openapi3.Schemas) (*openapi3.SchemaRef, error) {
	schema, err := GetSchema(m.Type.Name(), types, schemas)
	if err != nil {
		return nil, err
	}

	desc := m.GetComment()
	if desc == "" {
		desc = strings.Join(m.Docs, " ")
	}
	schema.Value.Description = desc

	schema.Value.Description = desc
	schema.Value.Deprecated = CheckDeprecated(m.Docs)
	return schema, nil
}

func GetSchema(typ string, types map[string]spec.DefineStruct, schemas openapi3.Schemas) (*openapi3.SchemaRef, error) {
	// map[[2]string]map[string][2]*Bar
	if strings.HasPrefix(typ, "map[") {
		valueType, err := GetMapValueType(typ)
		if err != nil {
			return nil, errors.WithMessagef(err, "TrimMapTypePrefix with type \"%s\"", typ)
		}
		valueSchema, err := GetSchema(valueType, types, schemas)
		if err != nil {
			return nil, err
		}

		return &openapi3.SchemaRef{
			Value: &openapi3.Schema{
				Type:     openapi3.TypeObject,
				Nullable: true,
				AdditionalProperties: openapi3.AdditionalProperties{
					Schema: valueSchema,
				},
			},
		}, nil
	} else if strings.HasPrefix(typ, "[]") {
		itemSchema, err := GetSchema(typ[2:], types, schemas)
		if err != nil {
			return nil, err
		}
		return &openapi3.SchemaRef{
			Value: &openapi3.Schema{
				Type:     openapi3.TypeArray,
				Nullable: true,
				Items:    itemSchema,
			},
		}, nil
	} else if typ[0] == '[' {
		i := 1
		for ; i < len(typ)-1; i++ {
			if typ[i] == ']' {
				break
			}
		}
		if typ[i] != ']' {
			return nil, ErrInvalidType
		}
		dimension, err := strconv.ParseUint(typ[1:i], 10, 64)
		if err != nil {
			return nil, errors.WithMessagef(err, "parse array \"%s\" dimension", typ)
		}
		itemSchema, err := GetSchema(typ[i+1:], types, schemas)
		if err != nil {
			return nil, err
		}
		return &openapi3.SchemaRef{
			Value: &openapi3.Schema{
				Type:     openapi3.TypeArray,
				Items:    itemSchema,
				MinItems: dimension,
				MaxItems: &dimension,
			},
		}, nil
	} else if typ[0] == '*' {
		elementSchema, err := GetSchema(typ[1:], types, schemas)
		if err != nil {
			return nil, err
		}
		elementSchema.Value.Nullable = true
		return elementSchema, nil
	}

	var openapiType, openapiFormat string
	switch typ {
	case "string":
		openapiType = openapi3.TypeString
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"byte", "rune", "uintptr":
		openapiType = openapi3.TypeInteger
		switch typ {
		case "byte":
			openapiFormat = constant.FormatUint8
		case "rune":
			openapiFormat = constant.FormatInt32
		case "uintptr":
			openapiFormat = constant.FormatUint
		default:
			openapiFormat = typ
		}
	case "float32", "float64":
		openapiType = openapi3.TypeNumber
		if typ == "float32" {
			openapiFormat = constant.FormatFloat
		} else {
			openapiFormat = constant.FormatDouble
		}
	case "bool":
		openapiType = openapi3.TypeBoolean
	case "interface{}", "any":
		openapiType = openapi3.TypeString
		openapiFormat = constant.FormatBinary
	default:
		if ds, ok := types[typ]; ok {
			return GetStructSchema(ds, types, schemas), nil
		}
		return nil, ErrInvalidType
	}
	return &openapi3.SchemaRef{
		Value: &openapi3.Schema{
			Type:   openapiType,
			Format: openapiFormat,
		},
	}, nil
}

func ParseTags(s *openapi3.SchemaRef, tags []*spec.Tag) bool {
	required := true
	s.Value.AllowEmptyValue = true

	for _, tag := range tags {
		switch tag.Key {
		case constant.TagKeyForm, constant.TagKeyJson:
			for _, opt := range tag.Options {
				if opt == constant.OptionOptional || opt == constant.OptionOmitempty {
					required = false
				} else if strings.HasPrefix(opt, constant.OptionDefault) {
					required = false
					FillDefault(s, opt[len(constant.OptionDefault)+1:])
				} else if strings.HasPrefix(opt, constant.OptionOptions) {
					FillEnumFromOptions(s, opt[len(constant.OptionOptions)+1:])
				} else if strings.HasPrefix(opt, constant.OptionRange) {
					FillMinMaxFromRange(s, opt[len(constant.OptionRange)+1:])
				}
			}
		case constant.TagKeyValidate:
			if ValidateContainOr(tag) {
				log.Println("currently parsing validate tag which contains \"|\" is not supported")
				continue
			}
			var (
				opt string
				// https://pkg.go.dev/github.com/go-playground/validator/v10#hdr-Dive
				inKeys bool
			)
			for i := -1; i < len(tag.Options); i++ {
				if i == -1 {
					opt = tag.Name
				} else {
					opt = tag.Options[i]
				}
				if opt == "keys" {
					inKeys = true
					continue
				}
				if opt == "endkeys" {
					inKeys = false
					continue
				}
				if inKeys {
					continue
				}
				if opt == "required" {
					if s.Value.Nullable {
						s.Value.Nullable = false
					} else {
						s.Value.AllowEmptyValue = false
					}
					continue
				}
				if opt == "dive" {
					if s.Value.Type == openapi3.TypeArray {
						s = s.Value.Items
					} else if s.Value.Type == openapi3.TypeObject {
						if s.Value.AdditionalProperties.Schema == nil {
							log.Println("invalid validate tag \"dive\" for non map type \"%s\"", s.Value.Title)
							return required
						}
						s = s.Value.AdditionalProperties.Schema
					}
				} else {
					ParseValidateOption(s, opt)
				}
			}
		}
	}
	return required
}

// https://pkg.go.dev/github.com/go-playground/validator/v10#hdr-Or_Operator
func ValidateContainOr(tag *spec.Tag) bool {
	var (
		opt    string
		inKeys bool
	)

	for i := -1; i < len(tag.Options); i++ {
		if i == -1 {
			opt = tag.Name
		} else {
			opt = tag.Options[i]
		}
		if opt == "keys" {
			inKeys = true
			continue
		}
		if opt == "endkeys" {
			inKeys = false
			continue
		}
		if inKeys {
			continue
		}

		if strings.Contains(opt, "|") {
			return true
		}
	}
	return false
}

func ParseValidateOption(s *openapi3.SchemaRef, opt string) error {
	kv := strings.SplitN(opt, "=", 2)
	if len(kv) != 2 {
		return nil
	}

	key := kv[0]
	value := kv[1]
	switch key {
	case "oneof":
		var es []string
		// oneof='red0x2Cgreen' 'blue0x2Cyellow'
		if strings.Contains(value, "'") {
			es = strings.Split(value, "' '")
			es[0] = strings.TrimPrefix(es[0], "'")
			es[len(es)-1] = strings.TrimSuffix(es[len(es)-1], "'")
		} else {
			es = strings.Split(value, " ")
		}
		for i, e := range es {
			es[i] = UnescapeValidateString(e)
		}

		enum := make([]interface{}, len(es))
		for i, e := range es {
			v, err := ParseValue(s.Value.Type, s.Value.Format, e)
			if err != nil {
				return err
			}
			enum[i] = v
		}
		s.Value.Enum = enum
	case "min", "gte", "gt":
		switch s.Value.Type {
		case openapi3.TypeInteger, openapi3.TypeNumber:
			var (
				min float64
				err error
			)
			if s.Value.Type == openapi3.TypeInteger {
				min, err = ParseInteger(s.Value.Format, value)
			} else {
				min, err = ParseNumber(s.Value.Format, value)
			}
			if err != nil {
				return err
			}
			if s.Value.Min == nil || *s.Value.Min < min {
				s.Value.Min = &min
				s.Value.ExclusiveMin = key == "gt"
			} else if *s.Value.Min == min && key == "gt" {
				s.Value.ExclusiveMin = true
			}
		case openapi3.TypeArray, openapi3.TypeString, openapi3.TypeObject:
			v, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return err
			}
			if key == "gt" {
				v++
			}
			switch s.Value.Type {
			case openapi3.TypeArray:
				s.Value.MinItems = v
			case openapi3.TypeString:
				s.Value.MinLength = v
			case openapi3.TypeObject:
				s.Value.MinProps = v
			}
		}
	case "max", "lte", "lt":
		switch s.Value.Type {
		case openapi3.TypeInteger, openapi3.TypeNumber:
			var (
				max float64
				err error
			)
			if s.Value.Type == openapi3.TypeInteger {
				max, err = ParseInteger(s.Value.Format, value)
			} else {
				max, err = ParseNumber(s.Value.Format, value)
			}
			if err != nil {
				return err
			}
			if s.Value.Max == nil || *s.Value.Max > max {
				s.Value.Max = &max
				s.Value.ExclusiveMax = key == "lt"
			} else if *s.Value.Max == max && key == "lt" {
				s.Value.ExclusiveMax = true
			}
		case openapi3.TypeArray, openapi3.TypeString, openapi3.TypeObject:
			v, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return err
			}
			if key == "lt" {
				v--
			}
			switch s.Value.Type {
			case openapi3.TypeArray:
				s.Value.MaxItems = &v
			case openapi3.TypeString:
				s.Value.MaxLength = &v
			case openapi3.TypeObject:
				s.Value.MaxProps = &v
			}
		}
	case "len", "eq":
		typ := s.Value.Type
		if typ == openapi3.TypeInteger || typ == openapi3.TypeNumber || (typ == openapi3.TypeString && key == "eq") {
			var (
				v   float64
				e   interface{}
				err error
			)
			if typ == openapi3.TypeInteger {
				v, err = ParseInteger(s.Value.Format, value)
				e = v
			} else if typ == openapi3.TypeNumber {
				v, err = ParseNumber(s.Value.Format, value)
				e = v
			} else {
				e = value
			}
			if err != nil {
				return err
			}
			s.Value.Enum = []interface{}{e}
		} else if typ == openapi3.TypeArray || typ == openapi3.TypeObject || (typ == openapi3.TypeString && key == "len") {
			v, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return err
			}
			if typ == openapi3.TypeArray {
				s.Value.MinItems = v
				s.Value.MaxItems = &v
			} else if typ == openapi3.TypeObject {
				s.Value.MinProps = v
				s.Value.MaxProps = &v
			} else {
				s.Value.MinLength = v
				s.Value.MaxLength = &v
			}
		}
	}
	return nil
}

func FillDefault(s *openapi3.SchemaRef, defaultString string) error {
	var (
		v   interface{}
		err error
	)
	switch s.Value.Type {
	case openapi3.TypeBoolean:
		v, err = strconv.ParseBool(defaultString)
	case openapi3.TypeInteger:
		v, err = ParseInteger(s.Value.Format, defaultString)
	case openapi3.TypeNumber:
		v, err = ParseNumber(s.Value.Format, defaultString)
	default:
		v = defaultString
	}

	if err == nil {
		s.Value.Default = v
	}
	return err
}

func FillEnumFromOptions(s *openapi3.SchemaRef, options string) error {
	if len(s.Value.Enum) != 0 {
		return nil
	}

	var (
		v   interface{}
		err error
	)

	opts := strings.Split(options[1:len(options)-1], ",")
	enum := make([]interface{}, len(opts))
	for i, opt := range opts {
		switch s.Value.Type {
		case openapi3.TypeBoolean:
			v, err = strconv.ParseBool(opt)
		case openapi3.TypeInteger:
			v, err = ParseInteger(s.Value.Format, opt)
		case openapi3.TypeNumber:
			v, err = ParseNumber(s.Value.Format, opt)
		default:
			v = opt
		}
		if err != nil {
			return err
		}
		enum[i] = v
	}
	s.Value.Enum = enum
	return nil
}

func FillMinMaxFromRange(s *openapi3.SchemaRef, rng string) error {
	var (
		min, max                   float64
		exclusiveMin, exclusiveMax bool
		err                        error
	)

	exclusiveMin = rng[0] == '('
	exclusiveMax = rng[len(rng)-1] == ')'
	parts := strings.Split(rng[1:len(rng)-1], ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid range value \"%s\"", rng)
	}
	if parts[0] != "" {
		switch s.Value.Type {
		case openapi3.TypeInteger:
			min, err = ParseInteger(s.Value.Format, parts[0])
			if err != nil {
				return err
			}
		case openapi3.TypeNumber:
			min, err = ParseNumber(s.Value.Format, parts[0])
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("range options is not valid for type \"%s\"", s.Value.Type)
		}
		if s.Value.Min == nil || *s.Value.Min < min {
			s.Value.Min = &min
			s.Value.ExclusiveMin = exclusiveMin
		} else if *s.Value.Min == min && exclusiveMin {
			s.Value.ExclusiveMin = true
		}
	}
	if parts[1] != "" {
		switch s.Value.Type {
		case openapi3.TypeInteger:
			max, err = ParseInteger(s.Value.Format, parts[1])
			if err != nil {
				return err
			}
		case openapi3.TypeNumber:
			max, err = ParseNumber(s.Value.Format, parts[1])
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("range options is not valid for type \"%s\"", s.Value.Type)
		}
		if s.Value.Max == nil || *s.Value.Max > max {
			s.Value.Max = &max
			s.Value.ExclusiveMax = exclusiveMax
		} else if *s.Value.Max == max && exclusiveMax {
			s.Value.ExclusiveMax = true
		}
	}
	return nil
}

func ParseValue(typ string, format string, s string) (interface{}, error) {
	switch typ {
	case openapi3.TypeBoolean:
		return strconv.ParseBool(s)
	case openapi3.TypeInteger:
		return ParseInteger(format, s)
	case openapi3.TypeNumber:
		return ParseNumber(format, s)
	case openapi3.TypeString:
		return s, nil
	default:
		return nil, fmt.Errorf("can't parse type \"%s\"")
	}
}

func ParseInteger(format string, s string) (float64, error) {
	var bits int

	switch format {
	case constant.FormatInt8, constant.FormatUint8:
		bits = 8
	case constant.FormatInt16, constant.FormatUint16:
		bits = 16
	case constant.FormatInt32, constant.FormatUint32:
		bits = 32
	default:
		bits = 64
	}
	if IsUint(format) {
		v, err := strconv.ParseUint(s, 10, bits)
		return float64(v), err
	}
	v, err := strconv.ParseInt(s, 10, bits)
	return float64(v), err
}

func ParseNumber(format string, s string) (float64, error) {
	if format == constant.FormatFloat {
		return strconv.ParseFloat(s, 32)
	}
	return strconv.ParseFloat(s, 64)
}

func GetFieldName(m spec.Member) string {
	name := m.Name
	if tagName, err := m.GetPropertyName(); err != nil {
		name = tagName
	}
	return name
}

func IsUint(format string) bool {
	return format == constant.FormatUint || format == constant.FormatUint8 || format == constant.FormatUint16 ||
		format == constant.FormatUint32 || format == constant.FormatUint64
}

// CheckDeprecated check Deprecated: comment
func CheckDeprecated(docs spec.Doc) bool {
	for _, doc := range docs {
		if strings.HasPrefix(doc, "Deprecated:") {
			return true
		}
	}
	return false
}
