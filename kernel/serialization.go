// Serialization helpers (doc 38 §3.2 rows 7–8): OpenAPI query-encoding
// styles (form / form_no_explode / space_delimited / pipe_delimited /
// deep_object), JSON bodies, multipart/form-data uploads (io.Reader parts),
// and base64. Percent-encoding uses the `encodeURIComponent` unreserved set
// so the Go, TS, and Python kernels emit byte-identical query strings for
// identical input. Query params are an ORDERED slice (Go maps carry no
// insertion order); object-valued params encode with codepoint-sorted keys —
// deterministic by construction.
//
// Vendored kernel file — imports nothing (stdlib only).

package kernel

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"math"
	"mime/multipart"
	"net/textproto"
	"slices"
	"strconv"
	"strings"
)

// QueryStyle is an OpenAPI serialization style for one query parameter.
type QueryStyle string

// The closed style vocabulary of the config's `query_styles:` mapping.
const (
	QueryStyleForm           QueryStyle = "form"
	QueryStyleFormNoExplode  QueryStyle = "form_no_explode"
	QueryStyleSpaceDelimited QueryStyle = "space_delimited"
	QueryStylePipeDelimited  QueryStyle = "pipe_delimited"
	QueryStyleDeepObject     QueryStyle = "deep_object"
)

// QueryParam is one query parameter: a scalar (string / bool / integer /
// float / json.Number), a []any of scalars, or a map[string]any of scalars.
// A nil Value is skipped. An empty Style means QueryStyleForm.
type QueryParam struct {
	Name          string
	Value         any
	Style         QueryStyle
	Serialization *ParameterSerialization
}

// `encodeURIComponent`'s unreserved characters beyond alphanumerics
// (RFC 3986 unreserved + `!'()*`).
const uriComponentSafe = "!'()*-._~"

func isURIComponentSafe(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') ||
		strings.IndexByte(uriComponentSafe, c) >= 0
}

func percentEncode(value string) string {
	var b strings.Builder
	for i := 0; i < len(value); i++ {
		c := value[i]
		if isURIComponentSafe(c) {
			b.WriteByte(c)
		} else {
			b.WriteString(fmt.Sprintf("%%%02X", c))
		}
	}
	return b.String()
}

// scalarString renders a scalar with JS `String()` semantics: booleans
// lowercase, integral floats without a fraction.
func scalarString(value any) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case bool:
		return strconv.FormatBool(v), nil
	case int:
		return strconv.Itoa(v), nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case int8, int16, int32, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprint(v), nil
	case float32:
		return formatFloat(float64(v))
	case float64:
		return formatFloat(v)
	case json.Number:
		return v.String(), nil
	default:
		return "", fmt.Errorf("kernel: unsupported query scalar type %T", value)
	}
}

func formatFloat(f float64) (string, error) {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return "", fmt.Errorf("kernel: non-finite query value %v", f)
	}
	if f == math.Trunc(f) && math.Abs(f) < 1e21 {
		return strconv.FormatFloat(f, 'f', -1, 64), nil
	}
	return strconv.FormatFloat(f, 'g', -1, 64), nil
}

func encodedPair(key string, value any) (string, error) {
	rendered, err := scalarString(value)
	if err != nil {
		return "", err
	}
	return percentEncode(key) + "=" + percentEncode(rendered), nil
}

func joinedPair(key string, values []any, separator string) (string, error) {
	encoded := make([]string, len(values))
	for i, value := range values {
		rendered, err := scalarString(value)
		if err != nil {
			return "", err
		}
		encoded[i] = percentEncode(rendered)
	}
	return percentEncode(key) + "=" + strings.Join(encoded, separator), nil
}

func encodeSequence(pairs *[]string, key string, values []any, style QueryStyle) error {
	separators := map[QueryStyle]string{
		QueryStyleFormNoExplode:  ",",
		QueryStyleSpaceDelimited: "%20",
		QueryStylePipeDelimited:  "%7C",
	}
	if separator, joined := separators[style]; joined {
		pair, err := joinedPair(key, values, separator)
		if err != nil {
			return err
		}
		*pairs = append(*pairs, pair)
		return nil
	}
	for index, value := range values {
		pairKey := key
		if style == QueryStyleDeepObject {
			pairKey = fmt.Sprintf("%s[%d]", key, index)
		}
		pair, err := encodedPair(pairKey, value)
		if err != nil {
			return err
		}
		*pairs = append(*pairs, pair)
	}
	return nil
}

func encodeMapping(pairs *[]string, key string, value map[string]any, style QueryStyle) error {
	props := slices.Sorted(maps.Keys(value))
	switch style {
	case QueryStyleForm, QueryStyleDeepObject:
		for _, prop := range props {
			pairKey := prop
			if style == QueryStyleDeepObject {
				pairKey = key + "[" + prop + "]"
			}
			pair, err := encodedPair(pairKey, value[prop])
			if err != nil {
				return err
			}
			*pairs = append(*pairs, pair)
		}
		return nil
	case QueryStyleFormNoExplode:
		flat := make([]any, 0, len(props)*2)
		for _, prop := range props {
			flat = append(flat, prop, value[prop])
		}
		pair, err := joinedPair(key, flat, ",")
		if err != nil {
			return err
		}
		*pairs = append(*pairs, pair)
		return nil
	default:
		return fmt.Errorf("kernel: query style %q does not support object values", style)
	}
}

// EncodeQuery encodes ordered query params. Style defaults to OpenAPI's
// `form` (explode): slices repeat the key, maps flatten to their property
// names. Param order follows the slice — deterministic for identical input.
func EncodeQuery(params []QueryParam) (string, error) {
	var pairs []string
	for _, param := range params {
		if param.Value == nil && param.Serialization == nil {
			continue
		}
		if param.Serialization != nil {
			encoded, err := SerializeParameter("query", param.Name, param.Value, *param.Serialization)
			if err != nil {
				return "", err
			}
			if encoded != "" {
				pairs = append(pairs, encoded)
			}
			continue
		}
		style := param.Style
		if style == "" {
			style = QueryStyleForm
		}
		var err error
		switch value := param.Value.(type) {
		case map[string]any:
			err = encodeMapping(&pairs, param.Name, value, style)
		case []any:
			err = encodeSequence(&pairs, param.Name, value, style)
		default:
			var pair string
			if pair, err = encodedPair(param.Name, value); err == nil {
				pairs = append(pairs, pair)
			}
		}
		if err != nil {
			return "", err
		}
	}
	return strings.Join(pairs, "&"), nil
}

// AppendQuery appends an encoded query onto a URL that may already carry one.
func AppendQuery(url string, query string) string {
	if query == "" {
		return url
	}
	if strings.Contains(url, "?") {
		return url + "&" + query
	}
	return url + "?" + query
}

// JSONBody encodes a JSON request body and returns it with its content
// type. Compact output mirrors `JSON.stringify` (no whitespace, no HTML
// escaping); struct key order is field order, map keys sort — deterministic.
func JSONBody(value any) ([]byte, string, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, "", fmt.Errorf("kernel: encode JSON body: %w", err)
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), "application/json", nil
}

// UploadPart is one file-ish multipart field value, streamed from an
// io.Reader.
type UploadPart struct {
	Content     io.Reader
	Filename    string
	ContentType string
}

// MultipartField is one multipart field: a scalar, an UploadPart, or a
// []any of either.
type MultipartField struct {
	Name  string
	Value any
}

// EncodeMultipart renders multipart fields into a `multipart/form-data`
// body and its boundary-stamped content type. The body is buffered so the
// retry loop can replay it (the boundary is runtime-unique by design, like
// idempotency keys — never hashed into builds).
func EncodeMultipart(fields []MultipartField) ([]byte, string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	for _, field := range fields {
		items, isList := field.Value.([]any)
		if !isList {
			items = []any{field.Value}
		}
		for _, item := range items {
			if err := writeMultipartItem(writer, field.Name, item); err != nil {
				return nil, "", err
			}
		}
	}
	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("kernel: finalize multipart body: %w", err)
	}
	return buf.Bytes(), writer.FormDataContentType(), nil
}

func writeMultipartItem(writer *multipart.Writer, name string, item any) error {
	part, isUpload := item.(UploadPart)
	if !isUpload {
		rendered, err := scalarString(item)
		if err != nil {
			return err
		}
		return writer.WriteField(name, rendered)
	}
	filename := part.Filename
	if filename == "" {
		filename = "upload"
	}
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename=%q`, name, filename))
	if part.ContentType != "" {
		header.Set("Content-Type", part.ContentType)
	}
	dest, err := writer.CreatePart(header)
	if err != nil {
		return fmt.Errorf("kernel: create multipart part %q: %w", name, err)
	}
	if _, err := io.Copy(dest, part.Content); err != nil {
		return fmt.Errorf("kernel: read multipart part %q: %w", name, err)
	}
	return nil
}

// EncodeBase64 base64-encodes raw bytes (RFC 4648 standard alphabet).
func EncodeBase64(raw []byte) string {
	return base64.StdEncoding.EncodeToString(raw)
}
