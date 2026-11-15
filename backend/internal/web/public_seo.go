package web

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"unicode"
)

const (
	seoHomePath    = "/home"
	seoMarketPath  = "/model-plaza"
	seoDocsPath    = "/docs"
	seoSettingsTTL = 15 * time.Second
)

type PublicSEOSettings struct {
	SiteName     string
	SiteLogo     string
	FrontendURL  string
	BackendMode  bool
	MarketPublic bool
	CustomHome   bool
	SiteSubtitle string
}

type PublicSEO struct {
	load           func(context.Context) (PublicSEOSettings, error)
	docs           func(context.Context, string) (string, string, bool)
	fallbackURL    string
	mu             sync.Mutex
	settings       PublicSEOSettings
	settingsLoaded bool
	settingsExpiry time.Time
}

func NewPublicSEO(load func(context.Context) (PublicSEOSettings, error), fallbackURL string, docs func(context.Context, string) (string, string, bool)) *PublicSEO {
	return &PublicSEO{load: load, fallbackURL: seoSiteURL(fallbackURL), docs: docs}
}

func (s *PublicSEO) Invalidate() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settingsExpiry = time.Time{}
}

func (s *PublicSEO) snapshot(ctx context.Context) (PublicSEOSettings, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Only site settings are shared. Locales, route metadata and distinction
	// between public and authenticated documents are resolved per request.
	if time.Now().Before(s.settingsExpiry) {
		return s.settings, s.settingsLoaded
	}
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	settings, err := s.load(ctx)
	if err != nil {
		// A temporary settings failure is noindex and retried quickly without
		// making every request wait on the same unavailable database.
		s.settings = PublicSEOSettings{FrontendURL: s.fallbackURL}
		s.settingsLoaded, s.settingsExpiry = false, time.Now().Add(time.Second)
		return s.settings, false
	}
	settings.FrontendURL = seoSiteURL(settings.FrontendURL)
	if settings.FrontendURL == "" {
		settings.FrontendURL = s.fallbackURL
	}
	if settings.SiteName = strings.TrimSpace(settings.SiteName); settings.SiteName == "" {
		settings.SiteName = "Sub2API"
	}
	s.settings, s.settingsExpiry = settings, time.Now().Add(seoSettingsTTL)
	s.settingsLoaded = true
	return settings, true
}

func (s *PublicSEO) PageMetadata(request *http.Request) (PageMetadata, bool) {
	settings, loaded := s.snapshot(request.Context())
	meta := PageMetadata{SiteURL: settings.FrontendURL, NoIndex: true}
	if !loaded || settings.BackendMode {
		return meta, true
	}
	language, validQuery := seoRequestLanguage(request.URL.RawQuery)
	meta.Language = language.tag
	if !validQuery {
		return meta, true
	}

	path := strings.TrimSuffix(request.URL.Path, "/")
	switch path {
	case "", seoHomePath:
		path = seoHomePath
		meta.Title, meta.Description = language.home.title, language.home.description
		if settings.CustomHome {
			meta.Title, meta.Description = "", strings.TrimSpace(settings.SiteSubtitle)
			meta.Language = ""
		}
	case "/model-market", seoMarketPath:
		if !settings.MarketPublic {
			return meta, true
		}
		path = seoMarketPath
		meta.Title, meta.Description = language.market.title, language.market.description
	case seoDocsPath:
		if s.docs == nil {
			return meta, true
		}
		meta.Title, meta.Description = language.docs.title, language.docs.description
	default:
		slug, found := strings.CutPrefix(path, seoDocsPath+"/")
		if !found || s.docs == nil || slug == "" || slug == "batch-image" || strings.Contains(slug, "/") {
			return meta, true
		}
		ctx, cancel := context.WithTimeout(request.Context(), time.Second)
		defer cancel()
		title, description, public := s.docs(ctx, slug)
		if !public {
			return meta, true
		}
		meta.Title, meta.Description = title, description
	}

	meta.NoIndex = false
	meta.Title = seoTitledPage(meta.Title, settings.SiteName)
	meta.OpenGraph = &seoOpenGraph{SiteName: settings.SiteName, Locale: language.openGraph, Type: "website"}
	if meta.Language == "" {
		meta.OpenGraph.Locale = ""
	}
	if strings.HasPrefix(path, seoDocsPath+"/") {
		// The article itself is not localized by the language of the UI.
		meta.OpenGraph.Locale = ""
		meta.OpenGraph.Type = "article"
	}
	meta.OpenGraph.Image = seoImageURL(settings.FrontendURL, settings.SiteLogo)
	if settings.FrontendURL == "" {
		return meta, true
	}
	variant := "en"
	if (path == seoHomePath && !settings.CustomHome) || path == seoMarketPath {
		variant = language.code
		meta.Alternates = seoAlternates(settings.FrontendURL, path)
	}
	meta.Canonical = seoPageURL(settings.FrontendURL, path, variant)
	meta.Structured = seoJSONLD(settings.SiteName, settings.FrontendURL, path, meta)
	return meta, true
}

func seoTitledPage(title, siteName string) string {
	if title == "" {
		return siteName
	}
	return title + " · " + siteName
}

type seoOpenGraph struct {
	SiteName string
	Locale   string
	Type     string
	Image    string
}

type seoAlternate struct {
	Lang string `xml:"hreflang,attr"`
	URL  string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
}

func seoAlternates(base, path string) []seoAlternate {
	result := make([]seoAlternate, 0, len(seoLanguages)+1)
	for _, lang := range seoLanguages {
		result = append(result, seoAlternate{Lang: lang.tag, URL: seoPageURL(base, path, lang.code), Rel: "alternate"})
	}
	return append(result, seoAlternate{Lang: "x-default", URL: seoPageURL(base, path, "en"), Rel: "alternate"})
}

func seoRequestLanguage(rawQuery string) (seoLanguage, bool) {
	query, err := url.ParseQuery(rawQuery)
	if err != nil {
		return seoLanguages[0], false
	}
	if len(query) == 0 {
		return seoLanguages[0], true
	}
	variants := query["lang"]
	if len(query) != 1 || len(variants) != 1 {
		return seoLanguages[0], false
	}
	for _, lang := range seoLanguages {
		if variants[0] == lang.code {
			return lang, true
		}
	}
	return seoLanguages[0], false
}

func seoSiteURL(value string) string {
	value = strings.TrimSpace(value)
	if strings.ContainsRune(value, '\\') || strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return ""
	}
	u, err := url.Parse(value)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" ||
		u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return ""
	}
	return strings.TrimRight(u.String(), "/")
}

func seoImageURL(base, value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "/") && !strings.HasPrefix(value, "//") && base != "" {
		origin, err := url.Parse(base)
		if err != nil {
			return ""
		}
		origin.Path, origin.RawPath = "", ""
		return seoSiteURL(origin.String() + value)
	}
	return seoSiteURL(value)
}

func seoPageURL(base, path, language string) string {
	value := base + path
	if language != "en" {
		value += "?lang=" + url.QueryEscape(language)
	}
	return value
}

func seoJSONLD(siteName, siteURL, path string, meta PageMetadata) []byte {
	siteID := siteURL + "/#website"
	website := map[string]any{
		"@type": "WebSite", "@id": siteID, "url": siteURL + "/home", "name": siteName,
	}
	pageType := "CollectionPage"
	switch {
	case path == seoHomePath:
		pageType = "WebPage"
	case strings.HasPrefix(path, seoDocsPath+"/"):
		pageType = "TechArticle"
	}
	page := map[string]any{
		"@type": pageType, "@id": meta.Canonical + "#webpage", "url": meta.Canonical, "name": meta.Title,
		"description": meta.Description, "isPartOf": map[string]string{"@id": siteID},
	}
	if pageType != "TechArticle" && meta.Language != "" {
		page["inLanguage"] = meta.Language
	}
	value, _ := json.Marshal(map[string]any{"@context": "https://schema.org", "@graph": []any{website, page}})
	return value
}

type seoSitemapEntry struct {
	URL        string         `xml:"loc"`
	Alternates []seoAlternate `xml:"xhtml:link"`
}

func (s *PublicSEO) Sitemap(w http.ResponseWriter, request *http.Request) {
	settings, loaded := s.snapshot(request.Context())
	sitemap := struct {
		XMLName xml.Name          `xml:"urlset"`
		XMLNS   string            `xml:"xmlns,attr"`
		XHTML   string            `xml:"xmlns:xhtml,attr"`
		Entries []seoSitemapEntry `xml:"url"`
	}{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		XHTML: "http://www.w3.org/1999/xhtml",
	}
	if loaded && !settings.BackendMode && settings.FrontendURL != "" {
		paths := []string{seoHomePath}
		if settings.MarketPublic {
			paths = append(paths, seoMarketPath)
		}
		for _, path := range paths {
			if path == seoHomePath && settings.CustomHome {
				sitemap.Entries = append(sitemap.Entries, seoSitemapEntry{URL: settings.FrontendURL + path})
				continue
			}
			alternates := seoAlternates(settings.FrontendURL, path)
			for _, lang := range seoLanguages {
				sitemap.Entries = append(sitemap.Entries, seoSitemapEntry{
					URL: seoPageURL(settings.FrontendURL, path, lang.code), Alternates: alternates,
				})
			}
		}
		if s.docs != nil {
			sitemap.Entries = append(sitemap.Entries, seoSitemapEntry{URL: settings.FrontendURL + seoDocsPath})
		}
	}
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	if request.Method == http.MethodHead {
		return
	}
	_, _ = fmt.Fprint(w, xml.Header)
	_ = xml.NewEncoder(w).Encode(sitemap)
}

func (s *PublicSEO) Robots(w http.ResponseWriter, request *http.Request) {
	settings, loaded := s.snapshot(request.Context())
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	if request.Method == http.MethodHead {
		return
	}
	// Crawl HTML marked noindex so search engines can observe the directive.
	// Vite assets and the public docs catalog remain accessible for rendering.
	_, _ = fmt.Fprint(w, "User-agent: *\nAllow: /\n")
	if loaded && !settings.BackendMode && settings.FrontendURL != "" {
		_, _ = fmt.Fprintf(w, "Sitemap: %s/sitemap.xml\n", settings.FrontendURL)
	}
}
