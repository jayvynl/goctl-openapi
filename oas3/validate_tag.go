package oas3

import (
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/zeromicro/go-zero/tools/goctl/api/spec"
)

// https://pkg.go.dev/github.com/go-playground/validator/v10#hdr-Or_Operator
func validateContainOr(tag *spec.Tag) bool {
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

func parseValidateOption(s *openapi3.SchemaRef, opt string) error {
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
