package mangadex

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sasmitai/comicdown/internal/provider"
	"golang.org/x/time/rate"
)

const (
	baseURL       = "https://api.mangadex.org"
	feedPageLimit = 100
	userAgent     = "comicdown/1.0 (github.com/sasmitai/comicdown)"
)

// Client is a rate-limited MangaDex API client implementing provider.Provider.
type Client struct {
	http          *http.Client
	limiter       *rate.Limiter
	atHomeLimiter *rate.Limiter
}

// NewClient returns a ready-to-use MangaDex API client.
func NewClient() *Client {
	return &Client{
		http:          &http.Client{Timeout: 30 * time.Second},
		limiter:       rate.NewLimiter(rate.Limit(5), 5),
		atHomeLimiter: rate.NewLimiter(rate.Limit(1), 1),
	}
}

// --- provider.Provider interface ---

func (c *Client) Name() string { return "MangaDex" }

func (c *Client) SupportsSearch() bool { return true }

func (c *Client) SupportedLanguages() []string {
	return []string{"en", "vi", "ja", "ko", "zh", "fr", "de", "es", "pt-br", "ru", "it", "id"}
}

func (c *Client) MatchURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return strings.Contains(u.Host, "mangadex.org")
}

func (c *Client) ExtractID(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL %q: %w", rawURL, err)
	}

	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	if len(parts) < 2 || parts[0] != "title" {
		return "", fmt.Errorf("URL %q is not a MangaDex title URL (expected /title/{id})", rawURL)
	}

	id := parts[1]
	if len(id) != 36 {
		return "", fmt.Errorf("extracted ID %q does not look like a UUID", id)
	}
	return id, nil
}

// Search searches manga by title using the MangaDex API.
func (c *Client) Search(ctx context.Context, query string, opts provider.SearchOpts) ([]provider.SearchResult, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 25
	}

	endpoint := fmt.Sprintf("%s/manga?title=%s&limit=%d&order[followedCount]=desc&includes[]=cover_art",
		baseURL, url.QueryEscape(query), limit)

	resp, err := c.get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("search manga: %w", err)
	}
	defer resp.Body.Close()

	var raw mangaListResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode search response: %w", err)
	}

	results := make([]provider.SearchResult, 0, len(raw.Data))
	for _, d := range raw.Data {
		sr := provider.SearchResult{
			ID:    d.ID,
			Title: extractTitle(d.Attributes.Title),
			Description: func() string {
				if v, ok := d.Attributes.Description["en"]; ok {
					return truncate(v, 200)
				}
				return ""
			}(),
			Status: d.Attributes.Status,
			Year:   d.Attributes.Year,
		}

		for _, tag := range d.Attributes.Tags {
			if name, ok := tag.Attributes.Name["en"]; ok {
				sr.Tags = append(sr.Tags, name)
			}
		}

		// Extract cover art filename
		for _, rel := range d.Relationships {
			if rel.Type == "cover_art" {
				sr.CoverURL = "https://uploads.mangadex.org/covers/" + d.ID + "/" + rel.ID + ".jpg"
			}
		}

		results = append(results, sr)
	}

	return results, nil
}

// FetchManga retrieves metadata for a single manga by ID.
func (c *Client) FetchManga(ctx context.Context, id string) (provider.Manga, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return provider.Manga{}, fmt.Errorf("rate limiter: %w", err)
	}

	endpoint := fmt.Sprintf("%s/manga/%s?includes[]=author&includes[]=artist&includes[]=cover_art", baseURL, id)
	resp, err := c.get(endpoint)
	if err != nil {
		return provider.Manga{}, fmt.Errorf("fetch manga %s: %w", id, err)
	}
	defer resp.Body.Close()

	var raw MangaResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return provider.Manga{}, fmt.Errorf("decode manga response: %w", err)
	}
	if raw.Result != "ok" {
		return provider.Manga{}, fmt.Errorf("manga API returned non-ok result for %s", id)
	}

	return parseManga(raw), nil
}

// FetchAllChapters retrieves all chapters for a manga in a given language.
func (c *Client) FetchAllChapters(ctx context.Context, mangaID, lang string) ([]provider.Chapter, error) {
	var all []provider.Chapter
	offset := 0

	for {
		if err := c.limiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limiter: %w", err)
		}

		page, total, err := c.fetchChapterPage(ctx, mangaID, lang, offset)
		if err != nil {
			return nil, err
		}
		all = append(all, page...)
		offset += len(page)

		if offset >= total || len(page) == 0 {
			break
		}
	}

	return all, nil
}

// FetchPageURLs retrieves the image CDN base URL and page filenames.
func (c *Client) FetchPageURLs(ctx context.Context, chapterID string) (provider.PageURLs, error) {
	if err := c.atHomeLimiter.Wait(ctx); err != nil {
		return provider.PageURLs{}, fmt.Errorf("at-home rate limiter: %w", err)
	}

	endpoint := fmt.Sprintf("%s/at-home/server/%s", baseURL, chapterID)
	resp, err := c.get(endpoint)
	if err != nil {
		return provider.PageURLs{}, fmt.Errorf("fetch at-home for chapter %s: %w", chapterID, err)
	}
	defer resp.Body.Close()

	var raw AtHomeResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return provider.PageURLs{}, fmt.Errorf("decode at-home response: %w", err)
	}
	if raw.Result != "ok" {
		return provider.PageURLs{}, fmt.Errorf("at-home API returned non-ok result for chapter %s", chapterID)
	}

	return buildPageURLs(raw), nil
}

// --- internal helpers ---

func (c *Client) fetchChapterPage(ctx context.Context, mangaID, lang string, offset int) ([]provider.Chapter, int, error) {
	endpoint := fmt.Sprintf(
		"%s/manga/%s/feed?translatedLanguage[]=%s&order[chapter]=asc&limit=%d&offset=%d",
		baseURL, mangaID, lang, feedPageLimit, offset,
	)
	resp, err := c.get(endpoint)
	if err != nil {
		return nil, 0, fmt.Errorf("fetch chapter feed (offset %d): %w", offset, err)
	}
	defer resp.Body.Close()

	var raw FeedResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, 0, fmt.Errorf("decode feed response: %w", err)
	}
	if raw.Result != "ok" {
		return nil, 0, fmt.Errorf("feed API returned non-ok result for manga %s", mangaID)
	}

	chapters := make([]provider.Chapter, 0, len(raw.Data))
	for _, d := range raw.Data {
		chapters = append(chapters, parseChapter(d))
	}
	return chapters, raw.Total, nil
}

func (c *Client) get(endpoint string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP GET %s: %w", endpoint, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, endpoint)
	}
	return resp, nil
}

// --- parsing ---

func parseManga(raw MangaResponse) provider.Manga {
	attr := raw.Data.Attributes

	title := extractTitle(attr.Title)
	desc := ""
	if v, ok := attr.Description["en"]; ok {
		desc = v
	}

	var altTitles []string
	for _, alt := range attr.AltTitles {
		for _, v := range alt {
			altTitles = append(altTitles, v)
		}
	}

	var tags []string
	for _, tag := range attr.Tags {
		if name, ok := tag.Attributes.Name["en"]; ok {
			tags = append(tags, name)
		}
	}

	var authorID, artistID, coverArtID, coverFileName string
	for _, rel := range raw.Data.Relationships {
		switch rel.Type {
		case "author":
			authorID = rel.ID
		case "artist":
			artistID = rel.ID
		case "cover_art":
			coverArtID = rel.ID
			if fn, ok := rel.Attributes["fileName"]; ok {
				if s, ok := fn.(string); ok {
					coverFileName = s
				}
			}
		}
	}

	return provider.Manga{
		ID:                          raw.Data.ID,
		Title:                       title,
		AltTitles:                   altTitles,
		Description:                 desc,
		Status:                      attr.Status,
		Year:                        attr.Year,
		OriginalLanguage:            attr.OriginalLanguage,
		AvailableTranslatedLanguages: attr.AvailableTranslatedLanguages,
		Tags:                        tags,
		AuthorID:                    authorID,
		ArtistID:                    artistID,
		CoverArtID:                  coverArtID,
		CoverFileName:               coverFileName,
	}
}

func parseChapter(d ChapterData) provider.Chapter {
	var groupID string
	for _, rel := range d.Relationships {
		if rel.Type == "scanlation_group" {
			groupID = rel.ID
		}
	}
	return provider.Chapter{
		ID:                 d.ID,
		Volume:             d.Attributes.Volume,
		Chapter:            d.Attributes.Chapter,
		Title:              d.Attributes.Title,
		TranslatedLanguage: d.Attributes.TranslatedLanguage,
		Pages:              d.Attributes.Pages,
		ScanlationGroupID:  groupID,
	}
}

func buildPageURLs(raw AtHomeResponse) provider.PageURLs {
	hash := raw.Chapter.Hash
	base := raw.BaseURL

	original := make([]string, len(raw.Chapter.Data))
	for i, f := range raw.Chapter.Data {
		original[i] = fmt.Sprintf("%s/data/%s/%s", base, hash, f)
	}

	saver := make([]string, len(raw.Chapter.DataSaver))
	for i, f := range raw.Chapter.DataSaver {
		saver[i] = fmt.Sprintf("%s/data-saver/%s/%s", base, hash, f)
	}

	return provider.PageURLs{
		Original: original,
		Saver:    saver,
		Hash:     hash,
	}
}

func extractTitle(t map[string]string) string {
	if v, ok := t["en"]; ok {
		return v
	}
	for _, v := range t {
		return v
	}
	return ""
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
