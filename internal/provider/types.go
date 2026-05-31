package provider

import "context"

// Manga represents manga metadata, normalized across all providers.
type Manga struct {
	ID                          string
	Title                       string
	AltTitles                   []string
	Description                 string
	Status                      string
	Year                        int
	OriginalLanguage            string
	AvailableTranslatedLanguages []string
	Tags                        []string
	AuthorID                    string
	ArtistID                    string
	CoverArtID                  string
	CoverFileName               string // Used to construct cover URL
}

// CoverURL returns the full cover art URL for MangaDex.
func (m Manga) CoverURL() string {
	if m.CoverArtID == "" || m.CoverFileName == "" {
		return ""
	}
	return "https://uploads.mangadex.org/covers/" + m.ID + "/" + m.CoverFileName
}

// Chapter represents a single chapter, normalized across all providers.
type Chapter struct {
	ID                 string
	Volume             string
	Chapter            string
	Title              string
	TranslatedLanguage string
	Pages              int
	ScanlationGroupID  string
}

// PageURLs contains the resolved image URLs for a chapter.
type PageURLs struct {
	Original []string // full-quality URLs
	Saver    []string // compressed URLs
	Hash     string
}

// SearchResult represents a single manga result from a search query.
type SearchResult struct {
	ID           string
	Title        string
	CoverURL     string
	Description  string
	Status       string
	Tags         []string
	ChapterCount int // 0 if unknown
	Year         int
}

// SearchOpts configures a search query.
type SearchOpts struct {
	Language string
	Limit    int
	Offset   int
}

// Provider is the interface that all manga sources must implement.
type Provider interface {
	// Identity
	Name() string
	MatchURL(rawURL string) bool
	ExtractID(rawURL string) (string, error)

	// Search
	Search(ctx context.Context, query string, opts SearchOpts) ([]SearchResult, error)

	// Manga details
	FetchManga(ctx context.Context, id string) (Manga, error)

	// Chapters
	FetchAllChapters(ctx context.Context, mangaID, lang string) ([]Chapter, error)

	// Images
	FetchPageURLs(ctx context.Context, chapterID string) (PageURLs, error)

	// Capabilities
	SupportsSearch() bool
	SupportedLanguages() []string
}
