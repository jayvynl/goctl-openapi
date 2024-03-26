package oas3

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/jayvynl/goctl-openapi/constant"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/tools/goctl/api/spec"
)

func getSchema(typ string, types map[string]spec.DefineStruct, schemas openapi3.Schemas) (*openapi3.SchemaRef, error) {
	// map[[2]string]map[string][2]*Bar
	if strings.HasPrefix(typ, "map[") {
		valueType, err := GetMapValueType(typ)
		if err != nil {
			return nil, errors.WithMessagef(err, "TrimMapTypePrefix with type \"%s\"", typ)
		}
		valueSchema, err := getSchema(valueType, types, schemas)
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
		itemSchema, err := getSchema(typ[2:], types, schemas)
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
		itemSchema, err := getSchema(typ[i+1:], types, schemas)
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
		elementSchema, err := getSchema(typ[1:], types, schemas)
		if err != nil {
			return nil, err
		}
		// pointer element is a struct type
		if elementSchema.Value == nil {
			// make a copy of original schema
			originalSchema := *schemas[typ[1:]]
			elementSchema = &originalSchema
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

func ParseTags(s *openapi3.SchemaRef, tags []*spec.Tag) (bool, bool) {
	required := true
	allowEmpty := true

	for _, tag := range tags {
		switch tag.Key {
		case constant.TagKeyForm, constant.TagKeyJson:
			for _, opt := range tag.Options {
				if s.Value == nil {
					if opt == constant.OptionOptional || opt == constant.OptionOmitempty ||
						strings.HasPrefix(opt, constant.OptionDefault) {
						required = false
					}
					continue
				}
				if opt == constant.OptionOptional || opt == constant.OptionOmitempty {
					required = false
				} else if strings.HasPrefix(opt, constant.OptionDefault) {
					required = false
					fillDefault(s, opt[len(constant.OptionDefault)+1:])
				} else if strings.HasPrefix(opt, constant.OptionOptions) {
					fillEnumFromOptions(s, opt[len(constant.OptionOptions)+1:])
				} else if strings.HasPrefix(opt, constant.OptionRange) {
					fillMinMaxFromRange(s, opt[len(constant.OptionRange)+1:])
				}
			}
		case constant.TagKeyValidate:
			if validateContainOr(tag) {
				fmt.Println("currently parsing validate tag which contains \"|\" is not supported")
				continue
			}
			var (
				opt string
				// https://pkg.go.dev/github.com/go-playground/validator/v10#hdr-Dive
				inKeys bool
			)
			for i := -1; i < len(tag.Options); i++ {
				if s.Value == nil {
					break
				}
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
						allowEmpty = false
					}
					continue
				}
				if opt == "dive" {
					if s.Value.Type == openapi3.TypeArray {
						s = s.Value.Items
					} else if s.Value.Type == openapi3.TypeObject {
						if s.Value.AdditionalProperties.Schema == nil {
							fmt.Printf("invalid validate tag \"dive\" for non map type \"%s\"\n", s.Value.Title)
							return required, allowEmpty
						}
						s = s.Value.AdditionalProperties.Schema
					}
				} else {
					parseValidateOption(s, opt)
				}
			}
		}
	}
	return required, allowEmpty
}

func GetStructSchema(typ spec.DefineStruct, types map[string]spec.DefineStruct, schemas openapi3.Schemas) *openapi3.SchemaRef {
	if _, ok := schemas[typ.Name()]; ok {
		return &openapi3.SchemaRef{Ref: fmt.Sprintf("#/components/schemas/%s", typ.Name())}
	}

	embeddedSchemas := make([]*openapi3.SchemaRef, 0)
	schema := &openapi3.SchemaRef{
		Value: &openapi3.Schema{
			Type:        openapi3.TypeObject,
			Title:       typ.Name(),
			Description: strings.Join([]string(typ.Docs), " "),
			Deprecated:  CheckDeprecated(typ.Docs),
			Properties:  make(openapi3.Schemas),
		},
	}

	// must set cache immediately, for breaking cycle type reference.
	schemas[typ.Name()] = schema
	for _, m := range typ.Members {
		fn := GetJsonFieldName(m)
		// is member a struct type?
		if mt, ok := m.Type.(spec.DefineStruct); ok {
			// embedded struct, recursive parse
			// currently go-zero does not support show members of nested struct over 2 levels(include).
			// we can get original type from type definitions in this case.
			mt = types[mt.Name()]
			ms := GetStructSchema(mt, types, schemas)
			if m.Name == "" {
				embeddedSchemas = append(embeddedSchemas, schemas[mt.Name()])
				continue
			}
			schema.Value.Properties[fn] = ms
		} else {
			memberSchema, err := getMemberSchema(m, types, schemas)
			if err != nil {
				fmt.Printf("invalid type of %s.%s\n", typ.Name(), m.Name)
				continue
			}
			schema.Value.Properties[fn] = memberSchema
		}

		if required, _ := ParseTags(schema.Value.Properties[fn], m.Tags()); required {
			schema.Value.Required = append(schema.Value.Required, fn)
		}
	}

	for _, embeddedSchema := range embeddedSchemas {
		for name, fieldSchema := range embeddedSchema.Value.Properties {
			schema.Value.Properties[name] = fieldSchema
		}
		schema.Value.Required = MergeRequired(schema.Value.Required, embeddedSchema.Value.Required)
	}
	return &openapi3.SchemaRef{Ref: fmt.Sprintf("#/components/schemas/%s", typ.Name())}
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

func GetJsonFieldName(m spec.Member) string {
	for _, tag := range m.Tags() {
		if tag.Key == constant.TagKeyJson {
			if tag.Name == "-" {
				return m.Name
			}
			return tag.Name
		}
	}
	return m.Name
}
