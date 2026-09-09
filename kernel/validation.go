// Structural wire validation precedes typed decoding, preserving required-field presence.
package kernel

import (
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// ValidationError identifies a contract failure without retaining input values.
type ValidationError struct {
	Path   string
	Reason string
}

func (e *ValidationError) Error() string {
	return "SDK validation failed at " + e.Path + ": " + e.Reason
}

// Contract carries a schema and its shared reference definitions.
type Contract struct {
	Schema      any
	Definitions map[string]any
}

// ParseContracts reads generator-authored contracts once at package initialization.
func ParseContracts(raw string) map[string]Contract {
	var registry struct {
		Definitions        map[string]any `json:"definitions"`
		RequestDefinitions map[string]any `json:"requestDefinitions"`
		RequestContracts   map[string]any `json:"requestContracts"`
		Contracts          map[string]any `json:"contracts"`
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	if decoder.Decode(&registry) != nil {
		panic("invalid generated validation registry")
	}
	out := make(map[string]Contract, len(registry.Contracts))
	for key, schema := range registry.Contracts {
		out[key] = Contract{schema, registry.Definitions}
	}
	for key, schema := range registry.RequestContracts {
		out[key] = Contract{schema, registry.RequestDefinitions}
	}
	return out
}

// Validate checks JSON-compatible values using bounded structural contracts.
func Validate(value any, contract Contract) error {
	if contract.Schema == nil {
		return nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return &ValidationError{"$", "not JSON-compatible"}
	}
	var wire any
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.UseNumber()
	if decoder.Decode(&wire) != nil {
		return &ValidationError{"$", "invalid JSON"}
	}
	budget := 100000
	return validateWire(wire, contract.Schema, contract.Definitions, "$", 0, &budget)
}

func validateWire(value, schema any, defs map[string]any, path string, depth int, budget *int) error {
	*budget--
	if depth > 64 || *budget < 0 {
		*budget = -1
		return &ValidationError{path, "validation limit exceeded"}
	}
	if flag, ok := schema.(bool); ok {
		if !flag {
			return &ValidationError{path, "value refused"}
		}
		return nil
	}
	shape, ok := schema.(map[string]any)
	if !ok {
		return &ValidationError{path, "invalid schema"}
	}
	if ref, ok := shape["$ref"].(string); ok {
		name := strings.TrimPrefix(ref, "#/$defs/")
		name = strings.ReplaceAll(strings.ReplaceAll(name, "~1", "/"), "~0", "~")
		target, exists := defs[name]
		if !strings.HasPrefix(ref, "#/$defs/") || !exists {
			return &ValidationError{path, "unresolved schema"}
		}
		if err := validateWire(value, target, defs, path, depth+1, budget); err != nil {
			return err
		}
	}
	if err := validateChoices(value, shape, defs, path, depth, budget); err != nil {
		return err
	}
	if err := validateConstants(value, shape, path); err != nil {
		return err
	}
	if kind, exists := shape["type"]; exists && !matchesType(value, kind) {
		return &ValidationError{path, "wrong type"}
	}
	if err := validateConstraints(value, shape, path, budget); err != nil {
		if validation, ok := err.(*ValidationError); ok && (strings.Contains(validation.Reason, "limit exceeded") || validation.Reason == "unsupported pattern" || validation.Reason == "invalid constraint") {
			*budget = -1
		}
		return err
	}
	if object, ok := value.(map[string]any); ok {
		return validateObject(object, shape, defs, path, depth, budget)
	}
	if items, ok := value.([]any); ok {
		if itemSchema, exists := shape["items"]; exists {
			for i, item := range items {
				if err := validateWire(item, itemSchema, defs, fmt.Sprintf("%s[%d]", path, i), depth+1, budget); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func validateChoices(value any, shape, defs map[string]any, path string, depth int, budget *int) error {
	if excluded, exists := shape["not"]; exists {
		if validateWire(value, excluded, defs, path, depth+1, budget) == nil {
			return &ValidationError{path, "excluded value"}
		}
		if *budget < 0 {
			return &ValidationError{path, "validation limit exceeded"}
		}
	}
	for _, keyword := range []string{"anyOf", "oneOf", "allOf"} {
		choices, ok := shape[keyword].([]any)
		if !ok {
			continue
		}
		matches := 0
		for _, choice := range choices {
			if validateWire(value, choice, defs, path, depth+1, budget) == nil {
				matches++
			}
		}
		if *budget < 0 {
			return &ValidationError{path, "validation limit exceeded"}
		}
		if matches == 0 || (keyword == "oneOf" && matches != 1) || (keyword == "allOf" && matches != len(choices)) {
			return &ValidationError{path, "union mismatch"}
		}
	}
	return nil
}

func matchesType(value, kind any) bool {
	if choices, ok := kind.([]any); ok {
		for _, choice := range choices {
			if matchesType(value, choice) {
				return true
			}
		}
		return false
	}
	switch kind {
	case "null":
		return value == nil
	case "string":
		_, ok := value.(string)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "number", "integer":
		number, ok := value.(json.Number)
		if !ok {
			return false
		}
		rational, valid := validationNumber(number)
		return valid && (kind == "number" || rational.IsInt())
	case "object":
		_, ok := value.(map[string]any)
		return ok
	case "array":
		_, ok := value.([]any)
		return ok
	default:
		return false
	}
}

func validateObject(value, shape, defs map[string]any, path string, depth int, budget *int) error {
	if required, ok := shape["required"].([]any); ok {
		for _, field := range required {
			if name, ok := field.(string); ok {
				if _, exists := value[name]; !exists {
					return &ValidationError{path + "." + name, "required field missing"}
				}
			}
		}
	}
	properties, _ := shape["properties"].(map[string]any)
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		schema, known := properties[key]
		childPath := path + "." + key
		if !known {
			schema, known = shape["additionalProperties"]
			childPath = path + ".*"
		}
		if known {
			if err := validateWire(value[key], schema, defs, childPath, depth+1, budget); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateConstants(value any, shape map[string]any, path string) error {
	if values, ok := shape["enum"].([]any); ok {
		found := false
		for _, candidate := range values {
			if equalConstant(value, candidate) {
				found = true
				break
			}
		}
		if !found {
			return &ValidationError{path, "unexpected enum value"}
		}
	}
	if constant, exists := shape["const"]; exists && !equalConstant(value, constant) {
		return &ValidationError{path, "unexpected constant"}
	}
	return nil
}

func equalConstant(left, right any) bool {
	a, aNumber := left.(json.Number)
	b, bNumber := right.(json.Number)
	if aNumber && bNumber {
		x, xOK := validationNumber(a)
		y, yOK := validationNumber(b)
		return xOK && yOK && x.Cmp(y) == 0
	}
	return reflect.DeepEqual(left, right)
}

func validationNumber(value json.Number) (*big.Rat, bool) {
	text := value.String()
	if len(text) > 1024 {
		return nil, false
	}
	if at := strings.IndexAny(text, "eE"); at >= 0 {
		exponent, err := strconv.ParseInt(text[at+1:], 10, 32)
		if err != nil || exponent < -1024 || exponent > 1024 {
			return nil, false
		}
	}
	return new(big.Rat).SetString(text)
}
