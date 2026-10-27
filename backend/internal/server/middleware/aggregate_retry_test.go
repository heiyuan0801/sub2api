package middleware

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/aggregateretry"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/stretchr/testify/require"
)

type aggregateAttemptContextKey struct{}

func TestAggregateRetryFreshAttemptsAndBillingIdentity(t *testing.T) {
	const original = `{"model":"public-model","messages":[]}`
	var ids []string
	attempts, released := 0, 0
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, attempts, released, "prior attempt resources must be released")
		attempts++
		defer func() { released++ }()
		state := aggregateretry.FromContext(r.Context())
		plan := state.Plan([]int64{10, 20})
		state.Select(plan[0])
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.Equal(t, original, string(body))
		require.Equal(t, "client", r.Header.Get("X-Original"))
		require.Equal(t, "/v1/responses", r.URL.Path)
		require.Nil(t, r.Context().Value(aggregateAttemptContextKey{}))
		state.Enable(r, body)
		id, ok := r.Context().Value(ctxkey.ClientRequestID).(string)
		require.True(t, ok)
		ids = append(ids, id)
		if attempts == 1 {
			// Model/URL/header/context rewrites must not escape this attempt.
			r.Header.Set("X-Original", "rewritten")
			r.URL.Path = "/rewritten"
			*r = *r.WithContext(context.WithValue(r.Context(), aggregateAttemptContextKey{}, true))
			w.Header().Set("Retry-After", "120")
			state.Retryable = true
			w.WriteHeader(502)
			_, _ = w.Write([]byte("failed A"))
			return
		}
		aggregateretry.SealBilling(r.Context())
		w.Header().Set("X-Client-Request-ID", ids[1])
		_, _ = w.Write([]byte("success B"))
	})
	r := httptest.NewRequest("POST", "/v1/responses", strings.NewReader(original))
	r.Header.Set("X-Original", "client")
	r.Header.Set("X-Client-Request-ID", "untrusted")
	w := httptest.NewRecorder()
	AggregateGroupRetry(next).ServeHTTP(w, r)
	require.Equal(t, 2, attempts)
	require.Equal(t, ids[0], ids[1])
	require.NotEqual(t, "untrusted", ids[0])
	require.Equal(t, "success B", w.Body.String())
	require.Empty(t, w.Header().Get("Retry-After"))
}

func TestAggregateRetryBoundaries(t *testing.T) {
	for _, mode := range []string{"all-failed", "no-candidate", "no-signal", "ordinary", "billing", "heartbeat", "partial", "large-error", "cancel", "bound-resource", "local-auth"} {
		t.Run(mode, func(t *testing.T) {
			attempts := 0
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attempts++
				s := aggregateretry.FromContext(r.Context())
				body, _ := io.ReadAll(r.Body)
				if mode != "ordinary" {
					s.Enable(r, body)
				}
				plan := s.Plan([]int64{1, 2})
				if attempts == 2 && mode == "no-candidate" {
					s.Exhausted = true
					w.WriteHeader(403)
					_, _ = w.Write([]byte("no candidate"))
					return
				}
				if attempts == 2 && mode == "local-auth" {
					w.WriteHeader(401)
					_, _ = w.Write([]byte("key revoked"))
					return
				}
				s.Select(plan[0])
				if mode == "heartbeat" {
					flusher, ok := w.(http.Flusher)
					require.True(t, ok)
					flusher.Flush()
				}
				if mode == "partial" {
					_, _ = w.Write([]byte("partial content"))
				}
				if mode == "billing" {
					aggregateretry.SealBilling(r.Context())
				}
				if mode != "no-signal" {
					s.Retryable = true
				}
				w.WriteHeader(502)
				if mode == "large-error" {
					_, _ = w.Write([]byte(strings.Repeat("x", 65537)))
				} else {
					_, _ = w.Write([]byte("upstream failure"))
				}
				if mode == "cancel" {
					cancel()
				}
			})
			body := `{"model":"gpt-5"}`
			if mode == "bound-resource" {
				body = `{"model":"gpt-5","previous_response_id":"resp_a"}`
			}
			w := httptest.NewRecorder()
			AggregateGroupRetry(next).ServeHTTP(w, httptest.NewRequest("POST", "/v1/responses", strings.NewReader(body)).WithContext(ctx))
			want := 1
			if mode == "all-failed" || mode == "no-candidate" || mode == "local-auth" {
				want = 2
			}
			require.Equal(t, want, attempts)
			if mode == "all-failed" || mode == "no-candidate" {
				require.Equal(t, 502, w.Code)
				require.Equal(t, "upstream failure", w.Body.String())
			}
			if mode == "local-auth" {
				require.Equal(t, 401, w.Code)
				require.Equal(t, "key revoked", w.Body.String())
			}
		})
	}
}

func TestAggregateRetryConcurrentIsolation(t *testing.T) {
	var wg sync.WaitGroup
	var ids sync.Map
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s := aggregateretry.FromContext(r.Context())
		body, _ := io.ReadAll(r.Body)
		s.Enable(r, body)
		plan := s.Plan([]int64{1, 2})
		s.Select(plan[0])
		if s.GroupID == 1 {
			s.Retryable = true
			w.WriteHeader(503)
			return
		}
		id, ok := r.Context().Value(ctxkey.ClientRequestID).(string)
		if !ok {
			t.Error("missing request ID")
			return
		}
		_, duplicate := ids.LoadOrStore(id, true)
		if duplicate {
			t.Error("request ID reused between clients")
		}
		aggregateretry.SealBilling(r.Context())
		_, _ = w.Write([]byte("ok"))
	})
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			AggregateGroupRetry(next).ServeHTTP(w, httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`{"model":"gpt-5"}`)))
			if w.Code != 200 || w.Body.String() != "ok" {
				t.Error("retry failed")
			}
		}()
	}
	wg.Wait()
}
