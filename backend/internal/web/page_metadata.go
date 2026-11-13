package web

import "context"

// PageMetadata customizes the initial HTML head for an SPA route.
type PageMetadata struct {
	Title       string
	Description string
	NoIndex     bool
}

type PageMetadataProvider func(context.Context, string) (PageMetadata, bool)
