// Pagination schemes (doc 38 §3.2 row 2) — one scheme value per
// `doctorine.sdk.yml` scheme: cursor / cursor_url / offset / page_number /
// token / single_page / item_cursor, consumed by Page/Pager (pager.go) so
// the advancing logic exists exactly once. Bindings mirror the config's
// role mappings verbatim; bodies are decoded JSON.
//
// Vendored kernel file — imports only sibling kernel files (stdlib only).

package kernel

import "encoding/json"

// PageScheme is the pure per-scheme logic Page and Pager delegate to.
type PageScheme interface {
	// Items extracts the page's item list from a response body.
	Items(body any) []any
	// NextRequest derives the next page's request, or ok == false when the
	// walked page is the last one.
	NextRequest(request PageRequest, body any) (next PageRequest, ok bool)
}

func nextFromCursor(request PageRequest, body any, requestCursor string, responseNextCursor string) (PageRequest, bool) {
	cursor, err := ReadBodyPointer(body, responseNextCursor)
	if err != nil || cursor == nil || cursor == "" {
		return request, false
	}
	switch cursor.(type) {
	case string, json.Number, float64, int, int64, float32:
		next, applyErr := ApplyRequestLocator(request, requestCursor, cursor)
		return next, applyErr == nil
	default:
		return request, false
	}
}

// CursorScheme: the response carries the next request's cursor value.
type CursorScheme struct {
	RequestCursor      string
	ResponseItems      string
	ResponseNextCursor string
}

// Items implements PageScheme.
func (s CursorScheme) Items(body any) []any { return ItemsAt(body, s.ResponseItems) }

// NextRequest implements PageScheme.
func (s CursorScheme) NextRequest(request PageRequest, body any) (PageRequest, bool) {
	return nextFromCursor(request, body, s.RequestCursor, s.ResponseNextCursor)
}

// TokenScheme: Google-style page tokens — cursor semantics, different role
// names.
type TokenScheme struct {
	RequestPageToken      string
	ResponseItems         string
	ResponseNextPageToken string
}

// Items implements PageScheme.
func (s TokenScheme) Items(body any) []any { return ItemsAt(body, s.ResponseItems) }

// NextRequest implements PageScheme.
func (s TokenScheme) NextRequest(request PageRequest, body any) (PageRequest, bool) {
	return nextFromCursor(request, body, s.RequestPageToken, s.ResponseNextPageToken)
}

// CursorURLScheme: the next page is a ready-made URL.
type CursorURLScheme struct {
	ResponseItems   string
	ResponseNextURL string
}

// Items implements PageScheme.
func (s CursorURLScheme) Items(body any) []any { return ItemsAt(body, s.ResponseItems) }

// NextRequest implements PageScheme.
func (s CursorURLScheme) NextRequest(request PageRequest, body any) (PageRequest, bool) {
	value, err := ReadBodyPointer(body, s.ResponseNextURL)
	nextURL, isString := value.(string)
	if err != nil || !isString || nextURL == "" {
		return request, false
	}
	// The next URL carries its own query string — clear the request's.
	request.URL = nextURL
	request.Query = nil
	return request, true
}

// OffsetScheme: the request offset advances by the page's item count.
type OffsetScheme struct {
	RequestOffset string
	ResponseItems string
	// ResponseTotal optionally stops the walk at a server-reported total
	// ("" = walk until an empty page).
	ResponseTotal string
}

// Items implements PageScheme.
func (s OffsetScheme) Items(body any) []any { return ItemsAt(body, s.ResponseItems) }

// NextRequest implements PageScheme.
func (s OffsetScheme) NextRequest(request PageRequest, body any) (PageRequest, bool) {
	count := len(s.Items(body))
	if count == 0 {
		return request, false
	}
	current, _ := ToFiniteNumber(ReadRequestLocator(request, s.RequestOffset))
	nextOffset := int(current) + count
	if s.ResponseTotal != "" {
		totalValue, err := ReadBodyPointer(body, s.ResponseTotal)
		if err == nil {
			if total, isNumber := ToFiniteNumber(totalValue); isNumber && float64(nextOffset) >= total {
				return request, false
			}
		}
	}
	next, err := ApplyRequestLocator(request, s.RequestOffset, nextOffset)
	return next, err == nil
}

// PageNumberScheme: the request page number increments.
type PageNumberScheme struct {
	RequestPage   string
	ResponseItems string
	// ResponseTotalPages optionally stops the walk ("" = walk until an
	// empty page).
	ResponseTotalPages string
}

// Items implements PageScheme.
func (s PageNumberScheme) Items(body any) []any { return ItemsAt(body, s.ResponseItems) }

// NextRequest implements PageScheme.
func (s PageNumberScheme) NextRequest(request PageRequest, body any) (PageRequest, bool) {
	if len(s.Items(body)) == 0 {
		return request, false
	}
	currentPage := 1
	if current, isNumber := ToFiniteNumber(ReadRequestLocator(request, s.RequestPage)); isNumber {
		currentPage = int(current)
	}
	nextPage := currentPage + 1
	if s.ResponseTotalPages != "" {
		totalValue, err := ReadBodyPointer(body, s.ResponseTotalPages)
		if err == nil {
			if totalPages, isNumber := ToFiniteNumber(totalValue); isNumber && float64(nextPage) > totalPages {
				return request, false
			}
		}
	}
	next, err := ApplyRequestLocator(request, s.RequestPage, nextPage)
	return next, err == nil
}

// SinglePageScheme is non-advancing: the one response IS the collection
// (iteration walks its items and stops — there is never a next page).
type SinglePageScheme struct {
	ResponseItems string
}

// Items implements PageScheme.
func (s SinglePageScheme) Items(body any) []any { return ItemsAt(body, s.ResponseItems) }

// NextRequest implements PageScheme.
func (s SinglePageScheme) NextRequest(request PageRequest, _ any) (PageRequest, bool) {
	return request, false
}

// ItemCursorScheme: the LAST item's field value is the next cursor.
type ItemCursorScheme struct {
	RequestCursor   string
	ResponseItems   string
	ResponseHasMore string
	// ItemCursorField is the item field whose last value advances the
	// cursor (`id`).
	ItemCursorField string
}

// Items implements PageScheme.
func (s ItemCursorScheme) Items(body any) []any { return ItemsAt(body, s.ResponseItems) }

// NextRequest implements PageScheme.
func (s ItemCursorScheme) NextRequest(request PageRequest, body any) (PageRequest, bool) {
	hasMore, err := ReadBodyPointer(body, s.ResponseHasMore)
	if err != nil || hasMore != true {
		return request, false
	}
	items := s.Items(body)
	if len(items) == 0 {
		return request, false
	}
	last, isRecord := items[len(items)-1].(map[string]any)
	if !isRecord {
		return request, false
	}
	switch cursor := last[s.ItemCursorField].(type) {
	case string, json.Number, float64, int, int64, float32:
		next, applyErr := ApplyRequestLocator(request, s.RequestCursor, cursor)
		return next, applyErr == nil
	default:
		return request, false
	}
}
