//go:build embed

package web

import (
	"bytes"
	"fmt"
	"html"
	"regexp"
	"strings"
)

var htmlLangAttribute = regexp.MustCompile(`\blang="[^"]*"`)

func injectPageMetadata(page []byte, meta PageMetadata, nonce string) []byte {
	if meta.Title != "" {
		start, end := bytes.Index(page, []byte("<title>")), bytes.Index(page, []byte("</title>"))
		if start >= 0 && end > start {
			page = bytes.Join([][]byte{page[:start+len("<title>")], []byte(html.EscapeString(meta.Title)), page[end:]}, nil)
		}
	}

	if meta.Language != "" {
		start := bytes.Index(page, []byte("<html"))
		if start >= 0 {
			end := start + bytes.IndexByte(page[start:], '>')
			if end > start {
				tag := page[start:end]
				value := []byte(`lang="` + html.EscapeString(meta.Language) + `"`)
				if htmlLangAttribute.Match(tag) {
					tag = htmlLangAttribute.ReplaceAllLiteral(tag, value)
				} else {
					tag = append(append(bytes.Clone(tag), ' '), value...)
				}
				page = bytes.Join([][]byte{page[:start], tag, page[end:]}, nil)
			}
		}
	}

	var tags strings.Builder
	writeSEOMeta(&tags, "name", "public-site-origin", meta.SiteURL)
	writeSEOMeta(&tags, "name", "description", meta.Description)
	if meta.NoIndex {
		writeSEOMeta(&tags, "name", "robots", "noindex")
	}
	if meta.Canonical != "" {
		fmt.Fprintf(&tags, `<link rel="canonical" href="%s" data-public-seo>`, html.EscapeString(meta.Canonical))
	}
	for _, alternate := range meta.Alternates {
		fmt.Fprintf(&tags, `<link rel="alternate" hreflang="%s" href="%s" data-public-seo>`,
			html.EscapeString(alternate.Lang), html.EscapeString(alternate.URL))
	}
	if graph := meta.OpenGraph; graph != nil {
		writeSEOMeta(&tags, "property", "og:type", graph.Type)
		writeSEOMeta(&tags, "property", "og:title", meta.Title)
		writeSEOMeta(&tags, "property", "og:description", meta.Description)
		writeSEOMeta(&tags, "property", "og:url", meta.Canonical)
		writeSEOMeta(&tags, "property", "og:site_name", graph.SiteName)
		writeSEOMeta(&tags, "property", "og:locale", graph.Locale)
		writeSEOMeta(&tags, "property", "og:image", graph.Image)
	}
	if len(meta.Structured) > 0 {
		fmt.Fprintf(&tags, `<script id="public-seo-jsonld" type="application/ld+json" nonce="%s" data-public-seo>%s</script>`,
			html.EscapeString(nonce), meta.Structured)
	}
	return bytes.Replace(page, []byte("</head>"), []byte(tags.String()+"</head>"), 1)
}

func writeSEOMeta(tags *strings.Builder, attribute, name, value string) {
	if value != "" {
		fmt.Fprintf(tags, `<meta %s="%s" content="%s" data-public-seo>`, attribute, name, html.EscapeString(value))
	}
}
