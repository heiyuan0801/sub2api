package middleware

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/pkg/aggregateretry"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
)

// AggregateGroupRetry re-enters the complete router with a fresh request/context
// only after a handler explicitly reports an eligible upstream failure. Returning
// from ServeHTTP first releases the failed attempt's slots and balance reservation.
func AggregateGroupRetry(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, inference := aggregateRouteSupported(r.Method, r.URL.Path)
		if !inference {
			next.ServeHTTP(w, r)
			return
		}
		state := &aggregateretry.State{}
		var body []byte
		var headers http.Header
		state.Capture = func(req *http.Request, original []byte) {
			body = bytes.Clone(original)
			headers = req.Header.Clone()
		}
		defer func() { state.Capture = nil }()
		// Never trust the caller's correlation header as a billing idempotency key.
		requestID := uuid.NewString()
		base := context.WithValue(r.Context(), ctxkey.ClientRequestID, requestID)
		base = aggregateretry.WithState(base, state)
		var last *aggregateAttemptWriter
		for attempt := 0; attempt < service.MaxAggregateGroups; attempt++ {
			if base.Err() != nil {
				return
			}
			state.BeginAttempt()
			req := r.Clone(base)
			if attempt > 0 {
				req.Header = headers.Clone()
				req.Body = http.NoBody
				if len(body) > 0 {
					req.Body = &replayBody{Reader: bytes.NewReader(body)}
				}
				req.ContentLength = int64(len(body))
				req.TransferEncoding = nil
			}
			out := &aggregateAttemptWriter{target: w, header: make(http.Header), state: state}
			next.ServeHTTP(out, req)
			if base.Err() != nil {
				return
			}
			if state.Exhausted && last != nil && !out.committed {
				last.commit()
				return
			}
			if out.committed || !state.Enabled || !state.Retryable || state.BillingSealed() || !state.HasNext() || attempt+1 == service.MaxAggregateGroups {
				out.commit()
				return
			}
			last = out
			slog.InfoContext(base, "aggregate_group_retry", "client_request_id", requestID, "failed_group_id", state.GroupID, "attempt", attempt+1)
		}
	})
}

type replayBody struct{ *bytes.Reader }

func (*replayBody) Close() error { return nil }

// Only a marked error is buffered, with a strict cap. Success/SSE writes pass
// straight through; even a flushed heartbeat permanently prohibits replay.
type aggregateAttemptWriter struct {
	target    http.ResponseWriter
	header    http.Header
	state     *aggregateretry.State
	status    int
	buffer    bytes.Buffer
	committed bool
}

func (w *aggregateAttemptWriter) Header() http.Header         { return w.header }
func (w *aggregateAttemptWriter) Unwrap() http.ResponseWriter { return w.target }
func (w *aggregateAttemptWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	if status < 400 || !w.state.Enabled || (!w.state.Retryable && !w.state.Exhausted) || w.state.BillingSealed() {
		w.commit()
	}
}
func (w *aggregateAttemptWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	if w.committed {
		return w.target.Write(body)
	}
	if w.buffer.Len()+len(body) > 64<<10 {
		w.commit()
		return w.target.Write(body)
	}
	return w.buffer.Write(body)
}
func (w *aggregateAttemptWriter) commit() {
	if w.committed {
		return
	}
	w.committed = true
	for name := range w.target.Header() {
		w.target.Header().Del(name)
	}
	for name, values := range w.header {
		w.target.Header()[name] = append([]string(nil), values...)
	}
	if w.status == 0 {
		w.status = http.StatusOK
	}
	w.target.WriteHeader(w.status)
	if w.buffer.Len() > 0 {
		_, _ = w.target.Write(w.buffer.Bytes())
		w.buffer.Reset()
	}
}
func (w *aggregateAttemptWriter) Flush() { _ = w.FlushError() }
func (w *aggregateAttemptWriter) FlushError() error {
	w.commit()
	return http.NewResponseController(w.target).Flush()
}
