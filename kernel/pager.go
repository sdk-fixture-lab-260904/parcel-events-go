// Page/Pager (doc 38 §3.2 row 2): one fetched page, plus the two idiomatic
// walk surfaces over ALL pages. The iterator shape mirrors the shipped
// Stainless Go SDKs (anthropic-sdk-go, cloudflare-go v2+): an auto-pager
// with Next()/Current()/Err() — except Next takes ctx explicitly
// (context-first; those SDKs capture the ctx at construction) — PLUS the
// Go 1.23+ range-over-func form (`for item, err := range pager.All(ctx)`).
// A single_page scheme walks its ONE response, so the iteration surface
// stays uniform across every list method.
//
// Vendored kernel file — imports only sibling kernel files (stdlib only).

package kernel

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
)

// PageFetcher is the client hook pages fetch through (Client implements it).
type PageFetcher interface {
	RequestPage(ctx context.Context, request PageRequest) (any, error)
}

// Page is one fetched page of T items.
type Page[T any] struct {
	// Request produced this page; Body is its decoded response body.
	Request PageRequest
	Body    any

	fetcher PageFetcher
	scheme  PageScheme
}

// FetchPage performs the first page fetch of a list call.
func FetchPage[T any](ctx context.Context, fetcher PageFetcher, request PageRequest, scheme PageScheme) (*Page[T], error) {
	body, err := fetcher.RequestPage(ctx, request)
	if err != nil {
		return nil, err
	}
	return NewPage[T](fetcher, request, body, scheme), nil
}

// NewPage wraps an already-fetched body (the generated method's seam).
func NewPage[T any](fetcher PageFetcher, request PageRequest, body any, scheme PageScheme) *Page[T] {
	return &Page[T]{Request: request, Body: body, fetcher: fetcher, scheme: scheme}
}

// RawItems is the page's item list as decoded JSON.
func (p *Page[T]) RawItems() []any { return p.scheme.Items(p.Body) }

// Items is the page's item list, cast to T.
func (p *Page[T]) Items() ([]T, error) {
	raw := p.RawItems()
	items := make([]T, len(raw))
	for i, item := range raw {
		cast, err := castItem[T](item)
		if err != nil {
			return nil, err
		}
		items[i] = cast
	}
	return items, nil
}

// NextPageRequest derives the next page's request (ok == false on the last
// page).
func (p *Page[T]) NextPageRequest() (PageRequest, bool) {
	return p.scheme.NextRequest(p.Request, p.Body)
}

// HasNextPage reports whether another page exists.
func (p *Page[T]) HasNextPage() bool {
	if len(p.RawItems()) == 0 {
		return false
	}
	_, ok := p.NextPageRequest()
	return ok
}

// GetNextPage fetches the next page; check HasNextPage first.
func (p *Page[T]) GetNextPage(ctx context.Context) (*Page[T], error) {
	request, ok := p.NextPageRequest()
	if !ok {
		return nil, fmt.Errorf("kernel: no next page; check HasNextPage first")
	}
	body, err := p.fetcher.RequestPage(ctx, request)
	if err != nil {
		return nil, err
	}
	return NewPage[T](p.fetcher, request, body, p.scheme), nil
}

// Pager returns an item iterator that transparently walks ALL pages
// starting from this one.
func (p *Page[T]) Pager() *Pager[T] {
	return &Pager[T]{page: p}
}

// Pager iterates items across page fetches:
//
//	pager := page.Pager()
//	for pager.Next(ctx) { use(pager.Current()) }
//	if err := pager.Err(); err != nil { ... }
type Pager[T any] struct {
	page    *Page[T]
	items   []T
	index   int
	started bool
	err     error
}

// Next advances to the next item, fetching the next page when the current
// one is exhausted. It returns false at the end of the collection or on
// error (check Err).
func (p *Pager[T]) Next(ctx context.Context) bool {
	if p.err != nil || p.page == nil {
		return false
	}
	if !p.started {
		p.started = true
		if p.items, p.err = p.page.Items(); p.err != nil {
			return false
		}
		p.index = -1
	}
	p.index++
	for p.index >= len(p.items) {
		if !p.page.HasNextPage() {
			return false
		}
		next, err := p.page.GetNextPage(ctx)
		if err != nil {
			p.err = err
			return false
		}
		p.page = next
		if p.items, p.err = p.page.Items(); p.err != nil {
			return false
		}
		p.index = 0
	}
	return true
}

// Current is the item Next advanced to.
func (p *Pager[T]) Current() T {
	return p.items[p.index]
}

// Err is the first fetch/decode error the walk hit (nil on clean exhaustion).
func (p *Pager[T]) Err() error { return p.err }

// All is the range-over-func form (Go 1.23+): yields every item across all
// pages; a fetch/decode error is yielded once as the final pair.
//
//	for item, err := range page.Pager().All(ctx) { ... }
func (p *Pager[T]) All(ctx context.Context) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		for p.Next(ctx) {
			if !yield(p.Current(), nil) {
				return
			}
		}
		if p.err != nil {
			var zero T
			yield(zero, p.err)
		}
	}
}

// castItem converts one decoded JSON item to T: a direct type assertion
// when possible, else a JSON round-trip into T (the generated models are
// json-tagged structs).
func castItem[T any](item any) (T, error) {
	if cast, ok := item.(T); ok {
		return cast, nil
	}
	var out T
	raw, err := json.Marshal(item)
	if err != nil {
		return out, fmt.Errorf("kernel: encode page item: %w", err)
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, fmt.Errorf("kernel: decode page item: %w", err)
	}
	return out, nil
}
