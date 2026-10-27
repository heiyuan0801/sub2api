// Package aggregateretry carries request-local state for aggregate group retries.
// Routing fields belong to the serving goroutine. Billing is sealed before any
// asynchronous usage task is submitted; no attempt can reopen that seal.
package aggregateretry

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"sync/atomic"

	"github.com/tidwall/gjson"
)

type contextKey struct{}

type State struct {
	Capture   func(*http.Request, []byte)
	Enabled   bool
	Retryable bool
	Exhausted bool
	GroupID   int64
	plan      []int64
	position  int
	billing   atomic.Bool
	// A client request consumes global user RPM once, even across groups.
	UserRPMCounted bool
}

func WithState(ctx context.Context, state *State) context.Context {
	return context.WithValue(ctx, contextKey{}, state)
}

func FromContext(ctx context.Context) *State {
	if ctx == nil {
		return nil
	}
	state, _ := ctx.Value(contextKey{}).(*State)
	return state
}

func SealBilling(ctx context.Context) {
	if state := FromContext(ctx); state != nil {
		state.billing.Store(true)
	}
}

func (s *State) BillingSealed() bool { return s.billing.Load() }

// Plan pins the original priority list. Re-reading authorization may remove
// entries, but editing a key cannot add destinations to an in-flight request.
func (s *State) Plan(ids []int64) []int64 {
	if s.plan == nil {
		s.plan = slices.Clone(ids)
		s.position = -1
	}
	return slices.Clone(s.plan[s.position+1:])
}

func (s *State) Select(id int64) {
	for i := s.position + 1; i < len(s.plan); i++ {
		if s.plan[i] == id {
			s.position = i
			s.GroupID = id
			return
		}
	}
}

func (s *State) HasNext() bool { return s.position+1 < len(s.plan) }

func (s *State) BeginAttempt() { s.Retryable = false; s.Exhausted = false }

func (s *State) Enable(r *http.Request, body []byte) {
	if s.Capture == nil {
		return
	}
	capture := s.Capture
	s.Capture = nil
	// Continuations and uploaded resources belong to their original upstream.
	// Full-history messages and inline cache hints remain portable.
	if !Portable(body) {
		return
	}
	capture(r, body)
	s.Enabled = true
}

func Portable(body []byte) bool {
	if !gjson.ValidBytes(body) {
		return false
	}
	var bound func(gjson.Result, int) bool
	bound = func(value gjson.Result, depth int) bool {
		if depth > 64 {
			return true
		}
		found := false
		value.ForEach(func(key, item gjson.Result) bool {
			switch strings.ToLower(key.String()) {
			case "previous_response_id", "conversation", "cachedcontent", "cached_content", "file_id", "fileid", "file_uri", "fileuri", "vector_store_ids", "container":
				if item.Type != gjson.Null && item.Raw != `""` && item.Raw != "[]" {
					found = true
				}
			}
			if !found && (item.IsObject() || item.IsArray()) {
				found = bound(item, depth+1)
			}
			return !found
		})
		return found
	}
	return !bound(gjson.ParseBytes(body), 0)
}
