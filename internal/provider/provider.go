package provider

import "context"

// Search performs a search on the given provider. Returns results or an error.
// This is a convenience function to avoid nil checks at call sites.
func Search(ctx context.Context, p Provider, query string, opts SearchOpts) ([]SearchResult, error) {
	if !p.SupportsSearch() {
		return nil, nil
	}
	return p.Search(ctx, query, opts)
}

// DetectProvider examines a URL and returns the matching provider from the list.
// Returns nil if no provider matches.
func DetectProvider(url string, providers []Provider) Provider {
	for _, p := range providers {
		if p.MatchURL(url) {
			return p
		}
	}
	return nil
}
