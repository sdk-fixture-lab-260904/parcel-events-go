package kernel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
)

// ParameterSerialization is the resolved OpenAPI style and explode contract.
type ParameterSerialization struct {
	Style         string
	Explode       bool
	AllowReserved bool
	ContentType   string
	OmitNil       bool
}

type serializedParameter struct {
	location, name string
	value          any
	serialization  ParameterSerialization
}

// WithParameter binds a path, header or cookie parameter before the request is sent.
func WithParameter(location, name string, value any, serialization ParameterSerialization) RequestOption {
	return func(r *requestConfig) {
		r.parameters = append(r.parameters, serializedParameter{location, name, value, serialization})
	}
}

// WithQuerySerialization binds query metadata without pre-encoding cursor values.
func WithQuerySerialization(name string, value any, serialization ParameterSerialization) RequestOption {
	return func(r *requestConfig) {
		r.query = append(r.query, QueryParam{Name: name, Value: value, Serialization: &serialization})
	}
}

func bindParameters(path string, config *requestConfig) (string, error) {
	cookies := []string{}
	for _, param := range config.parameters {
		encoded, err := SerializeParameter(param.location, param.name, param.value, param.serialization)
		if err != nil {
			return "", err
		}
		switch param.location {
		case "path":
			path = strings.ReplaceAll(path, "{"+param.name+"}", encoded)
		case "header":
			if _, exists := config.header[strings.ToLower(param.name)]; !exists {
				config.header[strings.ToLower(param.name)] = encoded
			}
		case "cookie":
			if encoded != "" {
				cookies = append(cookies, encoded)
			}
		default:
			return "", fmt.Errorf("kernel: unsupported parameter location")
		}
	}
	if config.header["cookie"] == "" && len(cookies) > 0 {
		config.header["cookie"] = strings.Join(cookies, "; ")
	}
	return path, nil
}

// SerializeParameter implements OpenAPI's defined scalar and flat collection styles.
func SerializeParameter(location, name string, value any, options ParameterSerialization) (string, error) {
	raw, _, err := JSONBody(value)
	if err != nil {
		return "", err
	}
	if options.OmitNil && string(raw) == "null" {
		return "", nil
	}
	if options.ContentType != "" {
		if options.ContentType != "application/json" && !strings.HasSuffix(options.ContentType, "+json") {
			return "", fmt.Errorf("kernel: unsupported parameter content type")
		}
		text := string(raw)
		if location == "header" {
			return cleanHeader(text)
		}
		if location == "path" {
			return parameterEscape(text, false), nil
		}
		return parameterEscape(name, false) + "=" + parameterEscape(text, location == "query" && options.AllowReserved), nil
	}
	var wire any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&wire); err != nil {
		return "", err
	}
	if wire == nil {
		return "", nil
	}
	encode := func(text string) string {
		if location == "header" {
			return text
		}
		return parameterEscape(text, location == "query" && options.AllowReserved)
	}
	parts, object, collection, err := parameterParts(wire, encode)
	if err != nil {
		return "", err
	}
	switch location {
	case "query":
		return queryParameter(name, parts, object, collection, options)
	case "path":
		return pathParameter(name, parts, object, collection, options)
	case "header":
		if options.Style != "simple" {
			return "", fmt.Errorf("kernel: unsupported header style")
		}
		return cleanHeader(strings.Join(explodedParts(parts, object, options.Explode), ","))
	case "cookie":
		if options.Style != "form" {
			return "", fmt.Errorf("kernel: unsupported cookie style")
		}
		return formParameter(name, parts, object, collection, options.Explode, "; "), nil
	default:
		return "", fmt.Errorf("kernel: unsupported parameter location")
	}
}

func cleanHeader(value string) (string, error) {
	if strings.ContainsAny(value, "\r\n") {
		return "", fmt.Errorf("kernel: invalid parameter header")
	}
	return value, nil
}

func parameterParts(value any, encode func(string) string) ([]string, bool, bool, error) {
	if object, ok := value.(map[string]any); ok {
		parts := []string{}
		for _, key := range slices.Sorted(maps.Keys(object)) {
			text, err := scalarString(object[key])
			if err != nil {
				return nil, false, false, err
			}
			parts = append(parts, encode(key), encode(text))
		}
		return parts, true, true, nil
	}
	if array, ok := value.([]any); ok {
		parts := []string{}
		for _, item := range array {
			text, err := scalarString(item)
			if err != nil {
				return nil, false, false, err
			}
			parts = append(parts, encode(text))
		}
		return parts, false, true, nil
	}
	text, err := scalarString(value)
	return []string{encode(text)}, false, false, err
}

func explodedParts(parts []string, object, explode bool) []string {
	if !object || !explode {
		return parts
	}
	result := []string{}
	for i := 0; i < len(parts); i += 2 {
		result = append(result, parts[i]+"="+parts[i+1])
	}
	return result
}

func formParameter(name string, parts []string, object, collection, explode bool, separator string) string {
	key := parameterEscape(name, false)
	if explode && collection {
		if object {
			return strings.Join(explodedParts(parts, true, true), separator)
		}
		pairs := []string{}
		for _, part := range parts {
			pairs = append(pairs, key+"="+part)
		}
		return strings.Join(pairs, separator)
	}
	return key + "=" + strings.Join(parts, ",")
}

func queryParameter(name string, parts []string, object, collection bool, options ParameterSerialization) (string, error) {
	switch options.Style {
	case "form":
		return formParameter(name, parts, object, collection, options.Explode, "&"), nil
	case "spaceDelimited", "pipeDelimited":
		if !collection || options.Explode {
			return "", fmt.Errorf("kernel: unsupported delimited parameter shape")
		}
		separator := "%20"
		if options.Style == "pipeDelimited" {
			separator = "%7C"
		}
		return parameterEscape(name, false) + "=" + strings.Join(parts, separator), nil
	case "deepObject":
		if !object || !options.Explode {
			return "", fmt.Errorf("kernel: unsupported deepObject parameter shape")
		}
		pairs := []string{}
		for i := 0; i < len(parts); i += 2 {
			pairs = append(pairs, parameterEscape(name, false)+"%5B"+parts[i]+"%5D="+parts[i+1])
		}
		return strings.Join(pairs, "&"), nil
	default:
		return "", fmt.Errorf("kernel: unsupported query style")
	}
}

func pathParameter(name string, parts []string, object, collection bool, options ParameterSerialization) (string, error) {
	switch options.Style {
	case "simple":
		return strings.Join(explodedParts(parts, object, options.Explode), ","), nil
	case "label":
		separator := ","
		if options.Explode {
			separator = "."
		}
		return "." + strings.Join(explodedParts(parts, object, options.Explode), separator), nil
	case "matrix":
		key := parameterEscape(name, false)
		if options.Explode && collection {
			if object {
				return ";" + strings.Join(explodedParts(parts, true, true), ";"), nil
			}
			result := ""
			for _, part := range parts {
				result += ";" + key + "=" + part
			}
			return result, nil
		}
		joined := strings.Join(parts, ",")
		if joined == "" {
			return ";" + key, nil
		}
		return ";" + key + "=" + joined, nil
	default:
		return "", fmt.Errorf("kernel: unsupported path style")
	}
}
