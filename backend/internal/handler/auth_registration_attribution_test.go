package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOAuthRegistrationAttributionKeepsCapturedCampaign(t *testing.T) {
	r := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(r)
	raw := `{"selected_source":"other","source_note":"free text","utm_source":"Google","utm_campaign":"first"}`
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/google/start?registration_attribution="+url.QueryEscape(raw), nil)
	captureOAuthRegistrationAttribution(c, true)
	cookies := r.Result().Cookies()
	require.Len(t, cookies, 1)
	cookie := cookies[0]
	require.True(t, cookie.HttpOnly && cookie.Secure)
	require.Equal(t, oauthPendingCookieMaxAgeSec, cookie.MaxAge)
	require.Equal(t, oauthPendingBrowserCookiePath, cookie.Path)

	r = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(r)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/google/callback", nil)
	c.Request.AddCookie(cookie)
	captured := readOAuthRegistrationAttribution(c)
	require.NotContains(t, captured, "free text")
	session := &dbent.PendingAuthSession{LocalFlowState: map[string]any{oauthRegistrationAttributionKey: captured}}
	body := json.RawMessage(`{"selected_source":"other","source_note":"论坛","utm_source":"forged"}`)
	ctx := oauthRegistrationContext(c, session, body)
	attribution, ok := service.RegistrationAttributionFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, "google", attribution.UTMSource)
	require.Equal(t, "first", attribution.UTMCampaign)
	require.Equal(t, "论坛", attribution.SourceNote)
}
