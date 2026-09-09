// Streaming readers (doc 38 §3.2 row 3, `streaming:` protocols sse|jsonl):
// a WHATWG-semantics SSE parser and a JSONL line reader, surfaced through
// Stream[T] — a single-use iterator of typed events with
// Next()/Current()/Err()/Close(). Generated code ships split
// Create/CreateStreaming method variants and only the streaming variant
// returns one of these. `[DONE]` (configurable) terminates SSE streams; an
// early Close releases the connection; an incomplete trailing SSE event is
// discarded per the spec. Line endings \r\n, \n, and lone \r all split.
//
// Vendored kernel file — imports only sibling kernel files (stdlib only).

package kernel

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// DefaultDoneSentinel is the SSE data payload that terminates a stream.
const DefaultDoneSentinel = "[DONE]"

// ServerSentEvent is one raw SSE event.
type ServerSentEvent struct {
	// Event is the event name ("message" when the server sent none).
	Event string
	// Data is the event payload (multi-line data joined with \n).
	Data string
	// ID is the last event id ("" when the server sent none).
	ID string
	// Retry is the server's reconnection delay in ms (-1 when unset).
	Retry int
}

// sseParser applies WHATWG EventSource field semantics, one line at a time.
// id/retry persist across events; event name and data reset per dispatch.
type sseParser struct {
	event string
	data  []string
	id    string
	retry int
	size  int
	err   error
	first bool
}

func newSSEParser() sseParser { return sseParser{retry: -1, first: true} }

func (p *sseParser) pushLine(line string) (ServerSentEvent, bool) {
	if p.first {
		line = strings.TrimPrefix(line, "\uFEFF")
		p.first = false
	}
	if line == "" {
		return p.dispatch()
	}
	if strings.HasPrefix(line, ":") {
		return ServerSentEvent{}, false
	}
	field, value := splitSSEField(line)
	switch field {
	case "event":
		p.event = value
	case "data":
		p.size += len(value) + 1
		if p.size > 8*1024*1024 {
			p.err = fmt.Errorf("kernel: stream event limit exceeded")
			return ServerSentEvent{}, false
		}
		p.data = append(p.data, value)
	case "id":
		if !strings.Contains(value, "\x00") {
			p.id = value
		}
	case "retry":
		if parsed, err := strconv.Atoi(value); err == nil && parsed >= 0 {
			p.retry = parsed
		}
	}
	return ServerSentEvent{}, false
}

func (p *sseParser) dispatch() (ServerSentEvent, bool) {
	if len(p.data) == 0 {
		p.event = ""
		return ServerSentEvent{}, false
	}
	name := p.event
	if name == "" {
		name = "message"
	}
	event := ServerSentEvent{Event: name, Data: strings.Join(p.data, "\n"), ID: p.id, Retry: p.retry}
	p.event = ""
	p.data = nil
	p.size = 0
	return event, true
}

func splitSSEField(line string) (field string, value string) {
	colon := strings.Index(line, ":")
	if colon == -1 {
		return line, ""
	}
	raw := line[colon+1:]
	return line[:colon], strings.TrimPrefix(raw, " ")
}

// scanEventLines splits on \r\n, lone \n, or lone \r (the WHATWG line
// grammar; bufio.ScanLines misses lone \r).
func scanEventLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexAny(data, "\r\n"); i >= 0 {
		if data[i] == '\n' {
			return i + 1, data[:i], nil
		}
		if i+1 < len(data) {
			if data[i+1] == '\n' {
				return i + 2, data[:i], nil
			}
			return i + 1, data[:i], nil
		}
		if atEOF {
			return i + 1, data[:i], nil
		}
		return 0, nil, nil // a trailing \r may be half of \r\n — need more data
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

type streamMode int

const (
	streamSSE streamMode = iota
	streamSSERaw
	streamJSONL
)

// Stream is a single-use iterator of typed streaming events:
//
//	defer stream.Close()
//	for stream.Next() { use(stream.Current()) }
//	if err := stream.Err(); err != nil { ... }
type Stream[T any] struct {
	body          io.ReadCloser
	eventContract Contract
	scanner       *bufio.Scanner
	mode          streamMode
	parser        sseParser
	doneSentinel  string
	current       T
	err           error
	done          bool
	closed        bool
}

func newStream[T any](body io.ReadCloser, mode streamMode, doneSentinel string) *Stream[T] {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	scanner.Split(scanEventLines)
	if doneSentinel == "" {
		doneSentinel = DefaultDoneSentinel
	}
	stream := &Stream[T]{body: body, scanner: scanner, mode: mode, parser: newSSEParser(), doneSentinel: doneSentinel}
	if validated, ok := body.(*validatedStreamBody); ok {
		stream.eventContract = validated.eventContract
	}
	return stream
}

// NewSSEStream reads typed events from an SSE body: JSON-parsed data,
// done-sentinel-aware ("" means DefaultDoneSentinel).
func NewSSEStream[T any](body io.ReadCloser, doneSentinel string) *Stream[T] {
	return newStream[T](body, streamSSE, doneSentinel)
}

// NewSSEEventStream reads raw SSE events (event name, id, retry preserved —
// no JSON parsing).
func NewSSEEventStream(body io.ReadCloser) *Stream[ServerSentEvent] {
	return newStream[ServerSentEvent](body, streamSSERaw, "")
}

// NewJSONLStream reads typed items from a JSONL body; blank lines skip.
func NewJSONLStream[T any](body io.ReadCloser) *Stream[T] {
	return newStream[T](body, streamJSONL, "")
}

func requireStreamable(response *http.Response, protocol streamMode) (io.ReadCloser, error) {
	if response.StatusCode == http.StatusNoContent {
		_ = response.Body.Close()
		return nil, NewConnectionError("response has no body to stream", nil)
	}
	media := strings.ToLower(strings.TrimSpace(strings.Split(response.Header.Get("Content-Type"), ";")[0]))
	supported := media == "text/event-stream"
	if protocol == streamJSONL {
		supported = media == "application/jsonl" || media == "application/x-ndjson" || media == "application/ndjson"
	}
	if !supported {
		_ = response.Body.Close()
		return nil, NewConnectionError("stream response content type mismatch", nil)
	}
	return response.Body, nil
}

// NewSSEStreamFromResponse wraps an ok response from Client.DoRaw ("" means
// DefaultDoneSentinel).
func NewSSEStreamFromResponse[T any](response *http.Response, doneSentinel string) (*Stream[T], error) {
	body, err := requireStreamable(response, streamSSE)
	if err != nil {
		return nil, err
	}
	return NewSSEStream[T](body, doneSentinel), nil
}

// NewJSONLStreamFromResponse wraps an ok response from Client.DoRaw.
func NewJSONLStreamFromResponse[T any](response *http.Response) (*Stream[T], error) {
	body, err := requireStreamable(response, streamJSONL)
	if err != nil {
		return nil, err
	}
	return NewJSONLStream[T](body), nil
}

// Next advances to the next event. It returns false at the end of the
// stream or on error (check Err); the body closes itself on either.
func (s *Stream[T]) Next() bool {
	if s.err != nil || s.done || s.closed {
		return false
	}
	for s.scanner.Scan() {
		event, dispatched := s.nextFromLine(s.scanner.Text())
		if s.err != nil || s.done || s.closed {
			s.finish()
			return false
		}
		if dispatched {
			s.current = event
			return true
		}
	}
	if scanErr := s.scanner.Err(); scanErr != nil {
		s.err = NewConnectionError("connection error while reading the stream", scanErr)
	}
	s.finish()
	return false
}

// nextFromLine feeds one line to the active protocol; dispatched is true
// when a typed event is ready in `event`.
func (s *Stream[T]) nextFromLine(line string) (event T, dispatched bool) {
	if s.mode == streamJSONL {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			return event, false
		}
		return s.decode([]byte(trimmed))
	}
	sse, ready := s.parser.pushLine(line)
	if s.parser.err != nil {
		s.err = s.parser.err
		return event, false
	}
	if !ready {
		return event, false
	}
	if s.mode == streamSSERaw {
		cast, _ := any(sse).(T) // T is ServerSentEvent by construction
		return cast, true
	}
	if sse.Data == s.doneSentinel {
		s.done = true
		return event, false
	}
	return s.decode([]byte(sse.Data))
}

func (s *Stream[T]) decode(raw []byte) (event T, dispatched bool) {
	if s.eventContract.Schema != nil {
		var wire any
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		if decoder.Decode(&wire) != nil {
			s.err = &ValidationError{"$", "invalid JSON"}
			return event, false
		}
		if err := Validate(wire, s.eventContract); err != nil {
			s.err = err
			return event, false
		}
	}
	if err := json.Unmarshal(raw, &event); err != nil {
		s.err = fmt.Errorf("kernel: decode streaming event: %w", err)
		return event, false
	}
	return event, true
}

// Current is the event Next advanced to.
func (s *Stream[T]) Current() T { return s.current }

// Err is the first read/decode error the stream hit (nil on clean end).
func (s *Stream[T]) Err() error { return s.err }

// Close releases the underlying connection; safe to call more than once
// (an early break must always be followed by Close, idiomatically deferred).
func (s *Stream[T]) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	return s.body.Close()
}

func (s *Stream[T]) finish() {
	s.done = true
	_ = s.Close()
}
