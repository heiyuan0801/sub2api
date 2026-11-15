package web

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPublicSEOPageMetadata(t *testing.T) {
	settings := PublicSEOSettings{
		SiteName: "Example", SiteLogo: "/logo.svg", FrontendURL: "https://gateway.example/api-site/", MarketPublic: true,
	}
	loads := 0
	seo := NewPublicSEO(func(context.Context) (PublicSEOSettings, error) {
		loads++
		return settings, nil
	}, "", func(_ context.Context, slug string) (string, string, bool) {
		return "Quickstart", "Connect your client", slug == "quickstart"
	})
	tests := []struct {
		path, title, canonical, lang string
		noIndex                      bool
		alternateCount               int
	}{
		{"/", "AI Model API Gateway · Example", "/home", "en", false, 6},
		{"/home?lang=zh-TW", "AI 模型 API 閘道 · Example", "/home?lang=zh-TW", "zh-Hant", false, 6},
		{"/model-plaza?lang=en", "Model Market · Example", "/model-plaza", "en", false, 6},
		{"/model-market?lang=ja", "モデルマーケット · Example", "/model-plaza?lang=ja", "ja", false, 6},
		{"/docs?lang=ko", "API 문서 · Example", "/docs", "ko", false, 0},
		{"/docs/quickstart", "Quickstart · Example", "/docs/quickstart", "en", false, 0},
		{path: "/model-plaza?embedded=1", noIndex: true},
		{path: "/model-plaza?search=deepseek&lang=zh", noIndex: true},
		{path: "/model-plaza?lang=de", noIndex: true},
		{path: "/model-plaza?lang=zh&lang=ko", noIndex: true},
		{path: "/docs/private-draft", noIndex: true},
		{path: "/docs/batch-image", noIndex: true},
		{path: "/admin/users", noIndex: true},
		{path: "/login", noIndex: true},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "http://untrusted.example"+test.path, nil)
			request.Header.Set("X-Forwarded-Host", "spoofed.example")
			request.Header.Set("X-Forwarded-Proto", "http")
			meta, found := seo.PageMetadata(request)
			require.True(t, found)
			require.Equal(t, test.noIndex, meta.NoIndex)
			require.Equal(t, "https://gateway.example/api-site", meta.SiteURL)
			require.Equal(t, test.title, meta.Title)
			require.Len(t, meta.Alternates, test.alternateCount)
			if test.noIndex {
				require.Empty(t, meta.Canonical)
				require.Nil(t, meta.OpenGraph)
				require.Empty(t, meta.Structured)
				return
			}
			require.Equal(t, test.lang, meta.Language)
			require.Equal(t, "https://gateway.example/api-site"+test.canonical, meta.Canonical)
			require.Equal(t, "https://gateway.example/logo.svg", meta.OpenGraph.Image)
			require.True(t, json.Valid(meta.Structured))
			require.NotContains(t, string(meta.Structured), "AggregateRating")
			require.NotContains(t, string(meta.Structured), "spoofed.example")
			if test.alternateCount > 0 {
				require.Equal(t, "x-default", meta.Alternates[len(meta.Alternates)-1].Lang)
			}
		})
	}
	require.Equal(t, 1, loads)
	settings.MarketPublic = false
	seo.Invalidate()
	meta, _ := seo.PageMetadata(httptest.NewRequest(http.MethodGet, "/model-plaza", nil))
	require.True(t, meta.NoIndex)

	settings.CustomHome, settings.SiteSubtitle = true, "Custom home summary"
	seo.Invalidate()
	customHome, _ := seo.PageMetadata(httptest.NewRequest(http.MethodGet, "/home?lang=zh", nil))
	require.False(t, customHome.NoIndex)
	require.Equal(t, "Example", customHome.Title)
	require.Equal(t, "Custom home summary", customHome.Description)
	require.Equal(t, "https://gateway.example/api-site/home", customHome.Canonical)
	require.Empty(t, customHome.Alternates)
	require.Empty(t, customHome.OpenGraph.Locale)

	settings.BackendMode = true
	seo.Invalidate()
	home, _ := seo.PageMetadata(httptest.NewRequest(http.MethodGet, "/home", nil))
	require.True(t, home.NoIndex)
	require.Empty(t, home.Title)
}

func TestPublicSEODiscovery(t *testing.T) {
	settings := PublicSEOSettings{FrontendURL: "https://gateway.example"}
	var settingErr error
	seo := NewPublicSEO(func(context.Context) (PublicSEOSettings, error) {
		return settings, settingErr
	}, "", func(_ context.Context, _ string) (string, string, bool) {
		return "", "", false
	})

	readSitemap := func(want int) {
		t.Helper()
		request := httptest.NewRequest(http.MethodGet, "https://untrusted.example/sitemap.xml", nil)
		request.Header.Set("X-Forwarded-Host", "spoofed.example")
		w := httptest.NewRecorder()
		seo.Sitemap(w, request)
		require.Equal(t, "application/xml; charset=utf-8", w.Header().Get("Content-Type"))
		result := struct {
			Name    xml.Name          `xml:"urlset"`
			Entries []seoSitemapEntry `xml:"url"`
		}{}
		require.NoError(t, xml.Unmarshal(w.Body.Bytes(), &result))
		require.Len(t, result.Entries, want)
		require.NotContains(t, w.Body.String(), "spoofed.example")
		require.NotContains(t, w.Body.String(), "embedded")
		if want != 0 {
			require.Contains(t, w.Body.String(), `xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"`)
			require.Contains(t, w.Body.String(), `xmlns:xhtml="http://www.w3.org/1999/xhtml"`)
			if settings.CustomHome {
				require.NotContains(t, w.Body.String(), "/home?lang=")
			} else {
				require.Contains(t, w.Body.String(), `<xhtml:link hreflang="zh-Hans" href="https://gateway.example/home?lang=zh" rel="alternate"></xhtml:link>`)
			}
			require.Equal(t, "https://gateway.example/home", result.Entries[0].URL)
			require.Equal(t, "https://gateway.example/docs", result.Entries[len(result.Entries)-1].URL)
		}
	}
	readRobots := func(wantSitemap bool) {
		t.Helper()
		w := httptest.NewRecorder()
		seo.Robots(w, httptest.NewRequest(http.MethodGet, "https://untrusted.example/robots.txt", nil))
		require.Equal(t, "text/plain; charset=utf-8", w.Header().Get("Content-Type"))
		require.Contains(t, w.Body.String(), "User-agent: *\nAllow: /\n")
		require.NotContains(t, w.Body.String(), "Disallow:")
		if wantSitemap {
			require.Contains(t, w.Body.String(), "Sitemap: https://gateway.example/sitemap.xml\n")
		} else {
			require.NotContains(t, w.Body.String(), "Sitemap:")
		}
	}
	readSitemap(6)
	readRobots(true)
	settings.MarketPublic = true
	seo.Invalidate()
	readSitemap(11)
	settings.CustomHome = true
	seo.Invalidate()
	readSitemap(7)
	settings.CustomHome = false
	settings.BackendMode = true
	seo.Invalidate()
	readSitemap(0)
	readRobots(false)

	settings.BackendMode = false
	settings.FrontendURL = "https://gateway.example?unsafe=x"
	seo.Invalidate()
	readSitemap(0)
	readRobots(false)
	meta, _ := seo.PageMetadata(httptest.NewRequest(http.MethodGet, "https://untrusted.example/home", nil))
	require.False(t, meta.NoIndex)
	require.Empty(t, meta.SiteURL)
	require.Empty(t, meta.Canonical)

	settingErr = errors.New("temporary database error")
	seo.Invalidate()
	readSitemap(0)
	readRobots(false)
	failed, _ := seo.PageMetadata(httptest.NewRequest(http.MethodGet, "/home?lang=zh", nil))
	require.True(t, failed.NoIndex)
}
