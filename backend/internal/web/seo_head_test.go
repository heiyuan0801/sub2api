//go:build embed

package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestEmbeddedPublicSEOHead(t *testing.T) {
	frontend, err := NewFrontendServer(&mockSettingsProvider{settings: map[string]string{"site_name": "Brand"}})
	require.NoError(t, err)
	seo := NewPublicSEO(func(context.Context) (PublicSEOSettings, error) {
		return PublicSEOSettings{SiteName: "Brand </script>", FrontendURL: "https://gateway.example", MarketPublic: true}, nil
	}, "", nil)
	r := gin.New()
	r.Use(frontend.Middleware(seo.PageMetadata))
	r.GET("/robots.txt", gin.WrapF(seo.Robots))
	r.GET("/sitemap.xml", gin.WrapF(seo.Sitemap))
	get := func(path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, http.StatusOK, w.Code)
		return w
	}

	zh := get("/model-plaza?lang=zh")
	require.Empty(t, zh.Header().Get("X-Robots-Tag"))
	require.Contains(t, zh.Body.String(), `<html lang="zh-Hans">`)
	require.Contains(t, zh.Body.String(), `<title>模型市场 · Brand &lt;/script&gt;</title>`)
	require.Contains(t, zh.Body.String(), `<link rel="canonical" href="https://gateway.example/model-plaza?lang=zh" data-public-seo>`)
	require.Contains(t, zh.Body.String(), `name="public-site-origin" content="https://gateway.example" data-public-seo`)
	require.Contains(t, zh.Body.String(), `property="og:title" content="模型市场 · Brand &lt;/script&gt;" data-public-seo`)
	require.Contains(t, zh.Body.String(), `id="public-seo-jsonld"`)
	require.Contains(t, zh.Body.String(), `\u003c/script\u003e`)

	en := get("/home")
	require.Contains(t, en.Body.String(), `<html lang="en">`)
	require.Contains(t, en.Body.String(), `<title>AI Model API Gateway · Brand &lt;/script&gt;</title>`)
	require.NotContains(t, en.Body.String(), `<title>模型市场`)
	for _, path := range []string{"/admin/users", "/model-plaza?embedded=1", "/docs"} {
		private := get(path)
		require.Equal(t, "noindex", private.Header().Get("X-Robots-Tag"))
		require.Contains(t, private.Body.String(), `<meta name="robots" content="noindex" data-public-seo>`)
		require.Contains(t, private.Body.String(), `name="public-site-origin" content="https://gateway.example" data-public-seo`)
		require.NotContains(t, private.Body.String(), `rel="canonical"`)
		require.NotContains(t, private.Body.String(), `id="public-seo-jsonld"`)
	}

	require.Contains(t, get("/robots.txt").Body.String(), "Sitemap: https://gateway.example/sitemap.xml")
	require.Contains(t, get("/sitemap.xml").Body.String(), `<loc>https://gateway.example/model-plaza</loc>`)
}
