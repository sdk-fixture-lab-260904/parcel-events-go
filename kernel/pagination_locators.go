// Pagination locator primitives (doc 38 §3.2 row 2) — the
// `doctorine.sdk.yml` role-mapping grammar shared by every page scheme:
// request locators (`query.cursor`, `body.page`, `header.x-next`) and
// `$`-rooted dotted response body pointers (`$.data`,
// `$.result_info.cursor`). Mirrors the TS kernel's pagination-locators.ts
// decision for decision; bodies are decoded JSON (map[string]any / []any).
//
// Vendored kernel file — imports only sibling kernel files (stdlib only).

package kernel

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// PageRequest is one page fetch: URL is a path relative to the client's
// base URL, or an absolute URL (the `cursor_url` scheme). Query is ordered
// (deterministic URLs for identical input).
type PageRequest struct {
	Method string
	URL    string
	Query  []QueryParam
	Header map[string]string
	Body   any
}

// ReadBodyPointer reads a `$`-rooted dotted body pointer (`$.data.items`).
// A missing segment yields nil; a malformed pointer is an error.
func ReadBodyPointer(body any, pointer string) (any, error) {
	if pointer != "$" && !strings.HasPrefix(pointer, "$.") {
		return nil, fmt.Errorf("kernel: invalid body pointer: %s", pointer)
	}
	current := body
	if pointer == "$" {
		return current, nil
	}
	for _, segment := range strings.Split(pointer[2:], ".") {
		record, isRecord := current.(map[string]any)
		if !isRecord {
			return nil, nil
		}
		current = record[segment]
	}
	return current, nil
}

func splitLocator(locator string) (kind string, rest string) {
	dot := strings.Index(locator, ".")
	if dot == -1 {
		return locator, ""
	}
	return locator[:dot], locator[dot+1:]
}

func deepSet(target any, path []string, value any) any {
	if len(path) == 0 {
		return value
	}
	base := map[string]any{}
	if record, isRecord := target.(map[string]any); isRecord {
		for key, existing := range record {
			base[key] = existing
		}
	}
	base[path[0]] = deepSet(base[path[0]], path[1:], value)
	return base
}

// ApplyRequestLocator immutably sets a request locator (`query.cursor` /
// `header.x-next` / `body.a.b`) to a scalar value.
func ApplyRequestLocator(request PageRequest, locator string, value any) (PageRequest, error) {
	kind, rest := splitLocator(locator)
	switch kind {
	case "query":
		request.Query = setQueryParam(request.Query, rest, value)
		return request, nil
	case "header":
		header := make(map[string]string, len(request.Header)+1)
		for name, existing := range request.Header {
			header[name] = existing
		}
		rendered, err := scalarString(value)
		if err != nil {
			return request, err
		}
		header[rest] = rendered
		request.Header = header
		return request, nil
	case "body":
		request.Body = deepSet(request.Body, strings.Split(rest, "."), value)
		return request, nil
	case "path":
		return request, fmt.Errorf("kernel: path.* locators cannot advance pagination at runtime")
	default:
		return request, fmt.Errorf("kernel: unknown request locator kind: %s", kind)
	}
}

// setQueryParam replaces a named param on a COPIED slice (or appends),
// preserving the original order.
func setQueryParam(query []QueryParam, name string, value any) []QueryParam {
	next := make([]QueryParam, len(query), len(query)+1)
	copy(next, query)
	for i := range next {
		if next[i].Name == name {
			next[i].Value = value
			return next
		}
	}
	return append(next, QueryParam{Name: name, Value: value})
}

// ReadRequestLocator reads the current value at a request locator (the
// offset / page_number schemes). Absent values yield nil.
func ReadRequestLocator(request PageRequest, locator string) any {
	kind, rest := splitLocator(locator)
	switch kind {
	case "query":
		for _, param := range request.Query {
			if param.Name == rest {
				return param.Value
			}
		}
		return nil
	case "header":
		if value, exists := request.Header[rest]; exists {
			return value
		}
		return nil
	case "body":
		current := request.Body
		for _, segment := range strings.Split(rest, ".") {
			record, isRecord := current.(map[string]any)
			if !isRecord {
				return nil
			}
			current = record[segment]
		}
		return current
	default:
		return nil
	}
}

// ItemsAt returns the item list at a response pointer (empty when absent /
// not a list / the pointer is malformed).
func ItemsAt(body any, pointer string) []any {
	value, err := ReadBodyPointer(body, pointer)
	if err != nil {
		return nil
	}
	items, isList := value.([]any)
	if !isList {
		return nil
	}
	return items
}

// ToFiniteNumber extracts a finite number from a number-or-numeric-string
// (decoded JSON numbers arrive as float64), else ok == false.
func ToFiniteNumber(value any) (number float64, ok bool) {
	switch v := value.(type) {
	case float64:
		return finiteNumber(v, true)
	case float32:
		return finiteNumber(float64(v), true)
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		parsed, err := v.Float64()
		return finiteNumber(parsed, err == nil)
	case string:
		parsed, err := strconv.ParseFloat(v, 64)
		return finiteNumber(parsed, err == nil)
	default:
		return 0, false
	}
}

func finiteNumber(number float64, ok bool) (float64, bool) {
	if !ok || math.IsNaN(number) || math.IsInf(number, 0) {
		return 0, false
	}
	return number, true
}
