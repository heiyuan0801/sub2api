package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/aggregateretry"
	"github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// Aggregate keys support stateless inference and discovery. Persistent task/resource
// endpoints require an ordinary key until their lifecycle ownership is supported.
func aggregateRouteSupported(method, path string) (metadata, inference bool) {
	if method == http.MethodGet && (strings.HasSuffix(path, "/models") || path == "/v1/usage" || path == "/v1/sub2api/billing") {
		return true, false
	}
	if (method == http.MethodPost) && (path == "/v1/responses" || path == "/responses" || path == "/openai/v1/responses" || path == "/backend-api/codex/responses") {
		return false, true
	}
	if method == http.MethodPost {
		switch path {
		case "/v1/chat/completions", "/v1/messages", "/v1/messages/count_tokens", "/antigravity/v1/messages", "/antigravity/v1/messages/count_tokens", "/antigravity/v1/chat/completions":
			return false, true
		}
		if strings.HasPrefix(path, "/v1beta/models/") || strings.HasPrefix(path, "/antigravity/v1beta/models/") {
			for _, suffix := range []string{":generateContent", ":streamGenerateContent", ":countTokens"} {
				if strings.HasSuffix(path, suffix) {
					return false, true
				}
			}
		}
	}
	return false, false
}

type aggregateGroupResolver interface {
	ResolveAggregateKey(context.Context, *service.APIKey, string, string, string) (*service.APIKey, error)
}

func prepareAggregateKey(c *gin.Context, s aggregateGroupResolver, cfg *config.Config, key *service.APIKey) (*service.APIKey, int, string) {
	if !key.IsAggregate {
		return key, 0, ""
	}
	metadata, inference := aggregateRouteSupported(c.Request.Method, c.Request.URL.Path)
	if !metadata && !inference {
		return nil, http.StatusBadRequest, "Aggregate keys support HTTP chat, Responses, Messages and Gemini inference; persistent media tasks and WebSocket sessions require an ordinary key"
	}
	if cfg != nil && cfg.RunMode == config.RunModeSimple {
		return nil, http.StatusBadRequest, "Aggregate keys require standard billing mode"
	}
	model, endpoint := "", ""
	if inference {
		model = groupModelAllowlistModelFromParams(c)
		if model == "" {
			candidates, ok := groupModelAllowlistModelsFromBody(c)
			if !ok {
				return nil, 0, ""
			} // The parser has already aborted the request.
			for _, candidate := range candidates {
				if model != "" && model != candidate {
					return nil, http.StatusBadRequest, "Conflicting model values"
				}
				model = candidate
			}
		}
		if strings.TrimSpace(model) == "" {
			return nil, http.StatusBadRequest, "model is required"
		}
		switch {
		case strings.Contains(c.Request.URL.Path, "/v1beta/"):
			endpoint = service.CompositeRouteEndpointGemini
		case strings.HasSuffix(c.Request.URL.Path, "/count_tokens"):
			endpoint = service.CompositeRouteEndpointCountTokens
		case strings.HasSuffix(c.Request.URL.Path, "/messages"):
			endpoint = service.CompositeRouteEndpointMessages
		case strings.HasSuffix(c.Request.URL.Path, "/responses"):
			endpoint = service.CompositeRouteEndpointResponses
		default:
			endpoint = service.CompositeRouteEndpointChatCompletions
		}
	}
	forced, _ := GetForcePlatformFromContext(c)
	selected, err := s.ResolveAggregateKey(c.Request.Context(), key, model, endpoint, forced)
	if err != nil {
		if errors.Is(err, service.ErrAggregateKeyUnavailable) || errors.Is(err, service.ErrAPIKeyNotFound) {
			return nil, http.StatusForbidden, "No available group supports this model; check selected groups, permissions and account models"
		}
		return nil, http.StatusServiceUnavailable, "Aggregate group selection is temporarily unavailable"
	}
	// The fresh key may carry stricter IP rules than the original auth snapshot.
	allowed, _ := ip.CheckIPRestrictionWithCompiledRules(ip.GetSecurityClientIP(c, cfg != nil && cfg.TrustForwardedIPForAPIKeyACL()), selected.CompiledIPWhitelist, selected.CompiledIPBlacklist)
	if !allowed {
		return nil, http.StatusForbidden, "Access denied"
	}
	if inference {
		if state := aggregateretry.FromContext(c.Request.Context()); state != nil && state.Capture != nil {
			// Gemini gets its model from the URL, so it may not have read the body yet.
			if _, ok := groupModelAllowlistModelsFromBody(c); !ok {
				return nil, 0, ""
			}
			body, _ := httputil.ReadRequestBodyWithPrealloc(c.Request)
			state.Enable(c.Request, body)
		}
	}
	return selected, 0, ""
}
