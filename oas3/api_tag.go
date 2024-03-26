package oas3

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

func fillDefault(s *openapi3.SchemaRef, defaultString string) error {
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

func fillEnumFromOptions(s *openapi3.SchemaRef, options string) error {
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

func fillMinMaxFromRange(s *openapi3.SchemaRef, rng string) error {
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
