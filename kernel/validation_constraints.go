package kernel

import (
	"encoding/json"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

func validateConstraints(value any, shape map[string]any, path string, budget *int) error {
	if number, ok := value.(json.Number); ok {
		return validateNumericConstraints(number, shape, path)
	}
	if text, ok := value.(string); ok {
		if format, ok := shape["format"].(string); ok && !matchesFormat(format, text) {
			return &ValidationError{path, "format " + format}
		}
		if err := validateSize(utf8.RuneCountInString(text), shape, "minLength", "maxLength", path); err != nil {
			return err
		}
		if pattern, ok := shape["pattern"].(string); ok {
			compiled, err := regexp.Compile(nativePattern(pattern))
			if err != nil {
				return &ValidationError{path, "unsupported pattern"}
			}
			if !compiled.MatchString(text) {
				return &ValidationError{path, "pattern mismatch"}
			}
		}
	}
	if object, ok := value.(map[string]any); ok {
		return validateSize(len(object), shape, "minProperties", "maxProperties", path)
	}
	if items, ok := value.([]any); ok {
		if err := validateSize(len(items), shape, "minItems", "maxItems", path); err != nil {
			return err
		}
		if shape["uniqueItems"] == true {
			seen := map[string]bool{}
			for _, item := range items {
				key, err := uniqueKey(item, 0, budget)
				if err != nil {
					return &ValidationError{path, "validation limit exceeded"}
				}
				if seen[key] {
					return &ValidationError{path, "duplicate item"}
				}
				seen[key] = true
			}
		}
	}
	return nil
}

func validateSize(size int, shape map[string]any, lower, upper, path string) error {
	number := new(big.Rat).SetInt64(int64(size))
	for _, key := range []string{lower, upper} {
		raw, exists := shape[key]
		if !exists {
			continue
		}
		bound, ok := raw.(json.Number)
		if !ok {
			return &ValidationError{path, "invalid constraint"}
		}
		limit, valid := validationNumber(bound)
		if !valid || (key == lower && number.Cmp(limit) < 0) || (key == upper && number.Cmp(limit) > 0) {
			return &ValidationError{path, key}
		}
	}
	return nil
}

func validateNumericConstraints(number json.Number, shape map[string]any, path string) error {
	value, valid := validationNumber(number)
	if !valid {
		return &ValidationError{path, "numeric validation limit exceeded"}
	}
	for _, key := range []string{"minimum", "maximum", "exclusiveMinimum", "exclusiveMaximum", "multipleOf"} {
		raw, exists := shape[key]
		if !exists {
			continue
		}
		bound, ok := raw.(json.Number)
		if !ok {
			return &ValidationError{path, "invalid constraint"}
		}
		limit, valid := validationNumber(bound)
		if !valid {
			return &ValidationError{path, "invalid constraint"}
		}
		comparison := value.Cmp(limit)
		rejected := (key == "minimum" && comparison < 0) || (key == "maximum" && comparison > 0) || (key == "exclusiveMinimum" && comparison <= 0) || (key == "exclusiveMaximum" && comparison >= 0)
		if key == "multipleOf" {
			rejected = limit.Sign() <= 0 || !new(big.Rat).Quo(value, limit).IsInt()
		}
		if rejected {
			return &ValidationError{path, key}
		}
	}
	return nil
}

// nativePattern spells the admitted ECMA-262 subset in Go's linear-time engine.
const ecmaWhitespace = " \t\n\v\f\r\u00a0\u1680\u2000-\u200a\u2028\u2029\u202f\u205f\u3000\ufeff"

func nativePattern(pattern string) string {
	var out strings.Builder
	inClass := false
	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		if c == '\\' && i+1 < len(pattern) {
			escaped, end := nativeEscape(pattern, i+1, inClass)
			out.WriteString(escaped)
			i = end
		} else if c == '.' && !inClass {
			out.WriteString("[^\\n\\r\\x{2028}\\x{2029}]")
		} else {
			out.WriteByte(c)
			if c == '[' {
				inClass = true
			}
			if c == ']' {
				inClass = false
			}
		}
	}
	return out.String()
}

func nativeEscape(pattern string, at int, inClass bool) (string, int) {
	escaped := pattern[at]
	replacement := map[byte]string{'d': "0-9", 'w': "A-Za-z0-9_", 'D': "^0-9", 'W': "^A-Za-z0-9_", 's': ecmaWhitespace, 'S': "^" + ecmaWhitespace}[escaped]
	if replacement != "" {
		if inClass {
			return replacement, at
		}
		return "[" + replacement + "]", at
	}
	if escaped == 'u' || escaped == 'x' {
		size := 4
		if escaped == 'x' {
			size = 2
		}
		if at+size >= len(pattern) {
			return "\\" + string(escaped), at
		}
		code, err := strconv.ParseInt(pattern[at+1:at+size+1], 16, 32)
		if err != nil {
			return "\\" + string(escaped), at
		}
		end := at + size
		scalar := rune(code)
		if code >= 0xd800 && code <= 0xdbff && end+6 < len(pattern) && pattern[end+1:end+3] == "\\u" {
			low, err := strconv.ParseInt(pattern[end+3:end+7], 16, 32)
			if err == nil {
				scalar = utf16.DecodeRune(scalar, rune(low))
				end += 6
			}
		}
		text := regexp.QuoteMeta(string(scalar))
		if text == "-" {
			text = "\\-"
		}
		return text, end
	}
	return "\\" + string(escaped), at
}
