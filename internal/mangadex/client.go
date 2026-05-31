package mangadex

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

const (
	baseURL        = "https://api.mangadex.org"
	feedPageLimit  = 100
	userAgent      = "comicdown/1.0 (github.com/sasmitai/comicdown)"
)

// Client is a rate-limited MangaDex API client.
type Client struct {
	http       *http.Client
	limiter    *rate.Limiter
	atHomeLimiter *rate.Limiter
}

// NewClient returns a ready-to-use MangaDex API client.
// General API: 5 req/s. At-home CDN: 1 req/s.
func NewClient() *Client {
	return &Client{
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
		limiter:       rate.NewLimiter(rate.Limit(5), 5),
		atHomeLimiter: rate.NewLimiter(rate.Limit(1), 1),
	}
}

// FetchManga retrieves metadata for a single manga by ID.
func (c *Client) FetchManga(id string) (Manga, error) {
	if err := c.limiter.Wait(context.Background()); err != nil {
		return Manga{}, fmt.Errorf("rate limiter: %w", err)
	}

	endpoint := fmt.Sprintf("%s/manga/%s", baseURL, id)
	resp, err := c.get(endpoint)
	if err != nil {
		return Manga{}, fmt.Errorf("fetch manga %s: %w", id, err)
	}
	defer resp.Body.Close()

	var raw MangaResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return Manga{}, fmt.Errorf("decode manga response: %w", err)
	}
	if raw.Result != "ok" {
		return Manga{}, fmt.Errorf("manga API returned non-ok result for %s", id)
	}

	return parseManga(raw), nil
}

// FetchAllChapters retrieves all chapters for a manga in a given language,
// paginating automatically through the feed.
func (c *Client) FetchAllChapters(mangaID, lang string) ([]Chapter, error) {
	var all []Chapter
	offset := 0

	for {
		if err := c.limiter.Wait(context.Background()); err != nil {
			return nil, fmt.Errorf("rate limiter: %w", err)
		}

		page, total, err := c.fetchChapterPage(mangaID, lang, offset)
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

// FetchPageURLs retrieves the image CDN base URL and page filenames
// for a chapter, using the at-home endpoint.
func (c *Client) FetchPageURLs(chapterID string) (PageURLs, error) {
	if err := c.atHomeLimiter.Wait(context.Background()); err != nil {
		return PageURLs{}, fmt.Errorf("at-home rate limiter: %w", err)
	}

	endpoint := fmt.Sprintf("%s/at-home/server/%s", baseURL, chapterID)
	resp, err := c.get(endpoint)
	if err != nil {
		return PageURLs{}, fmt.Errorf("fetch at-home for chapter %s: %w", chapterID, err)
	}
	defer resp.Body.Close()

	var raw AtHomeResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return PageURLs{}, fmt.Errorf("decode at-home response: %w", err)
	}
	if raw.Result != "ok" {
		return PageURLs{}, fmt.Errorf("at-home API returned non-ok result for chapter %s", chapterID)
	}

	return buildPageURLs(raw), nil
}

// ExtractMangaID parses a MangaDex manga URL and returns the UUID.
// Accepts both:
//   - https://mangadex.org/title/{uuid}
//   - https://mangadex.org/title/{uuid}/slug
func ExtractMangaID(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL %q: %w", rawURL, err)
	}

	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	// Expect: ["title", "{uuid}", ...]
	if len(parts) < 2 || parts[0] != "title" {
		return "", fmt.Errorf("URL %q is not a MangaDex title URL (expected /title/{id})", rawURL)
	}

	id := parts[1]
	if len(id) != 36 {
		return "", fmt.Errorf("extracted ID %q does not look like a UUID", id)
	}
	return id, nil
}

// --- internal helpers ---

func (c *Client) fetchChapterPage(mangaID, lang string, offset int) ([]Chapter, int, error) {
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

	chapters := make([]Chapter, 0, len(raw.Data))
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

func parseManga(raw MangaResponse) Manga {
	attr := raw.Data.Attributes

	// Prefer English title, fall back to first available.
	title := attr.Title["en"]
	if title == "" {
		for _, v := range attr.Title {
			title = v
			break
		}
	}

	desc := attr.Description["en"]

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

	var authorID, artistID, coverArtID string
	for _, rel := range raw.Data.Relationships {
		switch rel.Type {
		case "author":
			authorID = rel.ID
		case "artist":
			artistID = rel.ID
		case "cover_art":
			coverArtID = rel.ID
		}
	}

	return Manga{
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
	}
}

func parseChapter(d ChapterData) Chapter {
	var groupID, extURL string
	for _, rel := range d.Relationships {
		if rel.Type == "scanlation_group" {
			groupID = rel.ID
		}
	}
	if d.Attributes.ExternalURL != nil {
		extURL = *d.Attributes.ExternalURL
	}
	return Chapter{
		ID:                 d.ID,
		Volume:             d.Attributes.Volume,
		Chapter:            d.Attributes.Chapter,
		Title:              d.Attributes.Title,
		TranslatedLanguage: d.Attributes.TranslatedLanguage,
		Pages:              d.Attributes.Pages,
		PublishAt:          d.Attributes.PublishAt,
		ExternalURL:        extURL,
		ScanlationGroupID:  groupID,
	}
}

func buildPageURLs(raw AtHomeResponse) PageURLs {
	hash := raw.Chapter.Hash
	baseUrl := raw.BaseURL

	original := make([]string, len(raw.Chapter.Data))
	for i, f := range raw.Chapter.Data {
		original[i] = fmt.Sprintf("%s/data/%s/%s", baseUrl, hash, f)
	}

	saver := make([]string, len(raw.Chapter.DataSaver))
	for i, f := range raw.Chapter.DataSaver {
		saver[i] = fmt.Sprintf("%s/data-saver/%s/%s", baseUrl, hash, f)
	}

	return PageURLs{
		Original: original,
		Saver:    saver,
		Hash:     hash,
	}
}


