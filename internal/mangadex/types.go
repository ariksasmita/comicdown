package mangadex

import "time"

// --- Manga List (search) ---

type mangaListResponse struct {
	Result  string       `json:"result"`
	Data    []mangaData  `json:"data"`
	Limit   int          `json:"limit"`
	Offset  int          `json:"offset"`
	Total   int          `json:"total"`
}

// --- Manga ---

// MangaResponse is the top-level response for GET /manga/{id}.
type MangaResponse struct {
	Result string     `json:"result"`
	Data   mangaData  `json:"data"`
}

type mangaData struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		Title                      map[string]string   `json:"title"`
		AltTitles                  []map[string]string `json:"altTitles"`
		Description                map[string]string   `json:"description"`
		Status                     string              `json:"status"`
		Year                       int                 `json:"year"`
		ContentRating              string              `json:"contentRating"`
		OriginalLanguage           string              `json:"originalLanguage"`
		AvailableTranslatedLanguages []string          `json:"availableTranslatedLanguages"`
		Tags                       []struct {
			ID         string `json:"id"`
			Attributes struct {
				Name map[string]string `json:"name"`
			} `json:"attributes"`
		} `json:"tags"`
		Links map[string]string `json:"links"`
	} `json:"attributes"`
	Relationships []relationship `json:"relationships"`
}

// Relationship represents a related entity (author, artist, cover_art, etc.).
type relationship struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Attributes map[string]interface{} `json:"attributes"`
}

// --- Chapter Feed ---

// FeedResponse is the top-level response for GET /manga/{id}/feed.
type FeedResponse struct {
	Result string         `json:"result"`
	Data   []ChapterData  `json:"data"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
	Total  int            `json:"total"`
}

// ChapterData represents a single chapter from the feed.
type ChapterData struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		Volume             string    `json:"volume"`
		Chapter            string    `json:"chapter"`
		Title              string    `json:"title"`
		TranslatedLanguage string    `json:"translatedLanguage"`
		ExternalURL        *string   `json:"externalUrl"`
		Pages              int       `json:"pages"`
		PublishAt          time.Time `json:"publishAt"`
		ReadableAt         time.Time `json:"readableAt"`
	} `json:"attributes"`
	Relationships []struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	} `json:"relationships"`
}

// --- At-Home (Image URLs) ---

// AtHomeResponse is the response for GET /at-home/server/{chapterId}.
type AtHomeResponse struct {
	Result  string `json:"result"`
	BaseURL string `json:"baseUrl"`
	Chapter struct {
		Hash      string   `json:"hash"`
		Data      []string `json:"data"`
		DataSaver []string `json:"dataSaver"`
	} `json:"chapter"`
}
