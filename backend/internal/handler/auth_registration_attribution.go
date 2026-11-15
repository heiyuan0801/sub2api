package handler

import (
	"context"
	"encoding/json"
	"net/http"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	oauthRegistrationAttributionCookie  = "oauth_registration_attribution"
	oauthRegistrationAttributionKey     = "registration_attribution"
	oauthRegistrationAttributionFormKey = "oauth_registration_attribution_form"
)

func captureOAuthRegistrationAttribution(c *gin.Context, secure bool) {
	attribution := service.ParseRegistrationAttribution([]byte(c.Query(oauthRegistrationAttributionKey)))
	// Free text can contain personal details. Only accept it in request bodies
	// so OAuth start URLs and referrers never become a second source of it.
	attribution.SourceNote = ""
	if value, found := c.Get(oauthRegistrationAttributionFormKey); found {
		if form, ok := value.(service.RegistrationAttribution); ok {
			attribution = mergeOAuthRegistrationAttribution(attribution, form)
		}
	}
	if attribution == (service.RegistrationAttribution{}) {
		clearOAuthRegistrationAttribution(c, secure)
		return
	}
	body, err := json.Marshal(attribution)
	if err != nil || len(body) > service.MaxRegistrationAttributionBytes {
		clearOAuthRegistrationAttribution(c, secure)
		return
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name: oauthRegistrationAttributionCookie, Value: encodeCookieValue(string(body)),
		Path: oauthPendingBrowserCookiePath, MaxAge: oauthPendingCookieMaxAgeSec,
		HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode,
	})
}

func clearOAuthRegistrationAttribution(c *gin.Context, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name: oauthRegistrationAttributionCookie, Path: oauthPendingBrowserCookiePath,
		MaxAge: -1, HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode,
	})
}

func readOAuthRegistrationAttribution(c *gin.Context) string {
	raw, err := readCookieDecoded(c, oauthRegistrationAttributionCookie)
	if err != nil {
		return ""
	}
	return raw
}

func oauthRegistrationContext(c *gin.Context, pending *dbent.PendingAuthSession, submitted json.RawMessage) context.Context {
	raw := ""
	if pending == nil {
		raw = readOAuthRegistrationAttribution(c)
	} else {
		raw = pendingSessionStringValue(pending.LocalFlowState, oauthRegistrationAttributionKey)
	}
	attribution := mergeOAuthRegistrationAttribution(service.ParseRegistrationAttribution([]byte(raw)), service.ParseRegistrationAttribution(submitted))
	return service.WithRegistrationAttribution(c.Request.Context(), attribution)
}

func mergeOAuthRegistrationAttribution(attribution, selected service.RegistrationAttribution) service.RegistrationAttribution {
	if selected.SelectedSource != "" {
		attribution.SelectedSource = selected.SelectedSource
		attribution.SourceNote = selected.SourceNote
	}
	// Campaign fields captured when the OAuth flow started take precedence
	// over form metadata sent by a callback on the same browser.
	if attribution.UTMSource == "" && attribution.UTMMedium == "" && attribution.UTMCampaign == "" && attribution.UTMContent == "" && attribution.UTMTerm == "" {
		attribution.UTMSource = selected.UTMSource
		attribution.UTMMedium = selected.UTMMedium
		attribution.UTMCampaign = selected.UTMCampaign
		attribution.UTMContent = selected.UTMContent
		attribution.UTMTerm = selected.UTMTerm
	}
	return attribution
}
