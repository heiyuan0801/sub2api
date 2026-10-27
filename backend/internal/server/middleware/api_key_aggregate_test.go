package middleware

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
)

type aggregateResolverStub struct {
	model, endpoint, forced string
	calls                   int
}

func (s *aggregateResolverStub) ResolveAggregateKey(_ context.Context, key *service.APIKey, model, endpoint, forced string) (*service.APIKey, error) {
	s.calls++
	s.model = model
	s.endpoint = endpoint
	s.forced = forced
	out := *key
	id := int64(7)
	out.GroupID = &id
	return &out, nil
}
func TestAggregateRouteBoundaries(t *testing.T) {
	for _, p := range []string{"/v1/responses", "/responses", "/v1/messages", "/v1/messages/count_tokens", "/v1/chat/completions", "/v1beta/models/gemini:generateContent"} {
		_, ok := aggregateRouteSupported("POST", p)
		require.True(t, ok, p)
	}
	for _, p := range []string{"/v1/images/generations", "/v1/files", "/v1/batches", "/v1/responses/id", "/v1/videos"} {
		meta, ok := aggregateRouteSupported("POST", p)
		require.False(t, meta || ok, p)
	}
	_, ok := aggregateRouteSupported("GET", "/v1/responses")
	require.False(t, ok)
	for _, p := range []string{"/v1/models", "/v1beta/models", "/v1/usage", "/v1/sub2api/billing"} {
		meta, _ := aggregateRouteSupported("GET", p)
		require.True(t, meta)
	}
}
func TestAggregateModelParsingPreservesBodyAndRejectsConflicts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		body   string
		status int
	}{{`{"model":"gpt-5","messages":[]}`, 0}, {`{"model":"gpt-5","Model":"claude-sonnet-4-5"}`, 400}, {`{"model":"gpt-5","model":"claude-sonnet-4-5"}`, 400}, {`{}`, 400}} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(tc.body))
		c.Request.Header.Set("Content-Type", "application/json")
		resolver := &aggregateResolverStub{}
		selected, status, _ := prepareAggregateKey(c, resolver, &config.Config{}, &service.APIKey{ID: 8, IsAggregate: true})
		require.Equal(t, tc.status, status)
		if status == 0 {
			require.Equal(t, "gpt-5", resolver.model)
			require.Equal(t, service.CompositeRouteEndpointChatCompletions, resolver.endpoint)
			require.Equal(t, int64(8), selected.ID)
			body, err := io.ReadAll(c.Request.Body)
			require.NoError(t, err)
			require.Equal(t, tc.body, string(body))
		} else {
			require.Zero(t, resolver.calls)
		}
	}
}
func TestAggregateGeminiPathAndMetadata(t *testing.T) {
	for _, metadata := range []bool{true, false} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		resolver := &aggregateResolverStub{}
		if metadata {
			c.Request = httptest.NewRequest("GET", "/v1/models", nil)
		} else {
			c.Request = httptest.NewRequest("POST", "/v1beta/models/gemini-2.5-pro:generateContent", strings.NewReader(`{"contents":[]}`))
			c.Params = gin.Params{{Key: "modelAction", Value: "gemini-2.5-pro:generateContent"}}
		}
		_, code, _ := prepareAggregateKey(c, resolver, &config.Config{}, &service.APIKey{IsAggregate: true})
		require.Zero(t, code)
		if metadata {
			require.Empty(t, resolver.model)
		} else {
			require.Equal(t, "gemini-2.5-pro", resolver.model)
			require.Equal(t, service.CompositeRouteEndpointGemini, resolver.endpoint)
		}
	}
}
