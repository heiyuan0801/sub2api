package web

import "net/http"

// PageMetadata customizes the initial HTML head for an SPA route.
type PageMetadata struct {
	Title       string
	Description string
	NoIndex     bool
	Language    string
	SiteURL     string
	Canonical   string
	Alternates  []seoAlternate
	OpenGraph   *seoOpenGraph
	Structured  []byte
}

type PageMetadataProvider func(*http.Request) (PageMetadata, bool)
