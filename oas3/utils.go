package oas3

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	pathParamRe    = regexp.MustCompile(`/:([^/]+)`)
	ErrInvalidType = fmt.Errorf("invalid type")
)

func ConvertPath(path string) string {
	return pathParamRe.ReplaceAllString(path, `/{$1}`)
}

func GetProperty(properties map[string]string, key string) (string, error) {
	return strconv.Unquote(properties[key])
}

// https://pkg.go.dev/github.com/go-playground/validator/v10#hdr-Using_Validator_Tags
func UnescapeValidateString(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "0x2C", ","), "0x7C", "|")
}

// GetMapValueType map[[2]string][]string -> []string
func GetMapValueType(typ string) (string, error) {
	if !strings.HasPrefix(typ, "map[") {
		return "", ErrInvalidType
	}

	var level int
	for i := 4; i < len(typ)-1; i++ {
		if typ[i] == '[' {
			level++
		} else if typ[i] == ']' {
			if level == 0 {
				return typ[i+1:], nil
			}
			level--
		}
	}
	return "", ErrInvalidType
}

// MergeRequired merge schema required fields
func MergeRequired(rs ...[]string) []string {
	if len(rs) == 0 {
		return nil
	}
	if len(rs) == 1 {
		return rs[0]
	}

	merged := make([]string, len(rs[0]))
	copy(merged, rs[0])
	for i := 1; i < len(rs); i++ {
	out:
		for _, f := range rs[i] {
			for _, mf := range merged {
				if f == mf {
					continue out
				}
			}
			merged = append(merged, f)
		}
	}
	return merged
}
