# PLAN.md — comicdown Implementation Plan

## Overview

Terminal-based manga downloader: MangaDex API → download pages → optimize images → package as CBZ. Interactive TUI via Bubble Tea v2 with CLI fallback.

---

## Phase 1: Foundation ✅

- [x] Project scaffolding (`go mod init`, directory structure)
- [x] AGENTS.md (project context)
- [x] PLAN.md (this file)

## Phase 2: MangaDex API Client ✅

- [x] `internal/mangadex/types.go` — all API request/response structs
- [x] `internal/mangadex/client.go` — HTTP client with rate limiting
- [x] FetchManga, FetchAllChapters (paginated), FetchPageURLs, ExtractMangaID

## Phase 3: Core Pipeline (Headless) ✅

- [x] `internal/downloader/downloader.go` — concurrent, rate-limited, retry with backoff
- [x] `internal/optimizer/optimizer.go` — JPEG compress, resize, strip EXIF
- [x] `internal/packager/cbz.go` — CBZ zip + ComicInfo.xml
- [x] **Verified**: Full pipeline works end-to-end (Ch.5 downloaded → optimized → CBZ with metadata)

## Phase 4: TUI — Screen 1 (Input Form) ✅

- [x] `internal/tui/input.go` — URL, language, quality, max-width, output dir fields

## Phase 5: TUI — Screen 2 (Chapter Selector) ✅

- [x] `internal/tui/chapters.go` — checkbox list, select all/none, filter, scroll

## Phase 6: TUI — Screen 3 (Download Progress) ✅

- [x] `internal/tui/progress.go` — per-chapter progress bars, overall progress, abort

## Phase 7: TUI — Screen 4 (Summary) ✅

- [x] `internal/tui/summary.go` — results, file list, open folder, download more

## Phase 8: CLI Entry + Headless Mode ✅

- [x] `cmd/comicdown/main.go` — flag parsing, dual mode (TUI/CLI)

## Phase 9: Provider Interface + MangaDex Search

Refactor the MangaDex client behind a `Provider` interface so multiple sources can be supported.
Add search-by-title capability using the MangaDex `GET /manga?title=X` endpoint.

### Provider abstraction

- [ ] `internal/provider/provider.go` — `Provider` interface:
  - `Name() string`
  - `MatchURL(rawURL string) bool` — auto-detect provider from pasted URL
  - `ExtractID(rawURL string) (string, error)`
  - `Search(ctx, query string, opts SearchOpts) ([]SearchResult, error)`
  - `FetchManga(ctx, id string) (Manga, error)`
  - `FetchAllChapters(ctx, mangaID, lang string) ([]Chapter, error)`
  - `FetchPageURLs(ctx, chapterID string) (PageURLs, error)`
  - `SupportsSearch() bool`
  - `SupportedLanguages() []string`
- [ ] `internal/provider/types.go` — shared types: `Manga`, `Chapter`, `PageURLs`, `SearchResult`, `SearchOpts`
- [ ] Refactor `internal/mangadex/` to implement `Provider` interface
- [ ] Update all consumers (TUI, CLI) to use `Provider` instead of `mangadex.Client` directly

### MangaDex search

- [ ] `internal/mangadex/client.go` — add `Search(query, lang string, limit int) ([]provider.SearchResult, error)`
  - Calls `GET /manga?title={query}&limit={limit}&order[followedCount]=desc`
  - Returns `[]SearchResult` with ID, Title, CoverURL, Description, Status, Tags
- [ ] Cover art URL construction: `https://uploads.mangadex.org/covers/{mangaId}/{coverFileName}`

### CLI search mode

- [ ] `cmd/comicdown/main.go` — add `--search <query>` flag
  - `--search "One Piece" --provider mangadex --lang en`
  - Prints numbered list of results
  - User picks a number → continues to download flow

### Key decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Provider as interface | Not struct with switch | Clean extensibility — add MangaKakalot later without touching existing code |
| Shared types in provider/ | Not in mangadex/ | All providers return the same types; TUI/CLI don't care about the source |
| MangaDex search first | Before MangaKakalot | It's a clean REST API call, no scraping needed |
| Cover art from API | uploads.mangadex.org | MangaDex exposes cover filenames via relationships |

---

## Phase 10: TUI Search + Home Screen

Add a new home screen with two entry modes: paste URL (existing) or search by title (new).
When searching, show a results list the user can pick from, then flow into the chapter selector.

### TUI screen changes

- [ ] `internal/tui/home.go` — **New Screen 0: Home**
  - Two options: "🔗 Paste URL" and "🔍 Search by Title"
  - Arrow keys + Enter to pick
  - Selecting "Paste URL" → transitions to existing input form (Screen 1)
  - Selecting "Search by Title" → transitions to search form (Screen 0a)
  - Default screen when launching TUI with no `--url` flag

- [ ] `internal/tui/search.go` — **New Screen 0a: Search Form**
  - Provider selector (dropdown): `MangaDex` (initially only option)
  - Title text input with paste support
  - Language selector
  - [Search] button → calls provider.Search()
  - Spinner while fetching

- [ ] `internal/tui/results.go` — **New Screen 0b: Search Results**
  - Scrollable list of manga results from provider
  - Each row shows: cover thumbnail (placeholder), title, status, tags, chapter count
  - Enter on a result → fetches full manga + chapters → transitions to chapter selector (Screen 2)
  - Esc → back to search form
  - Filter by typing

- [ ] `internal/tui/app.go` — update screen routing
  - Add `screenHome`, `screenSearch`, `screenResults` to screen enum
  - New transitions: home→input, home→search, search→results, results→chapters
  - `Provider` stored on root Model, passed down to chapters/progress screens
  - URL auto-detection: if pasted URL matches a provider, skip home and go directly to chapters

### Updated TUI flow

```
Screen 0: HOME
  ├── [Paste URL]  → Screen 1: INPUT (existing)
  │                     ↓
  │                  Screen 2: CHAPTERS (existing)
  │                     ↓
  │                  Screen 3: PROGRESS (existing)
  │                     ↓
  │                  Screen 4: SUMMARY (existing)
  │
  └── [Search]    → Screen 0a: SEARCH FORM
                       ↓
                    Screen 0b: RESULTS LIST
                       ↓ (select one)
                    Screen 2: CHAPTERS (reuse existing)
                       ↓
                    ...continues as before
```

### Key decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Home as default screen | Not input form | Users can discover both modes naturally |
| Provider stored on Model | Not passed per-screen | Single source of truth; all screens use the same provider instance |
| Results screen separate from search | Not combined | Clean separation; results can be large and scrollable |
| MangaKakalot later | Not in this phase | Focus on getting search + TUI working with MangaDex first |

---

## Phase 11: Polish & Testing

- [ ] Unit tests for mangadex client (mock HTTP)
- [ ] Unit tests for optimizer (test images)
- [ ] Unit tests for CBZ packaging
- [ ] Unit tests for provider interface (mock provider)
- [ ] Integration test: full pipeline with a small chapter
- [ ] Error handling: network failures, invalid URLs, disk full
- [ ] Graceful shutdown on Ctrl+C (clean up temp files)
- [ ] Cross-platform: path handling for Windows/macOS/Linux

---

## Key Decisions Log

| Decision | Choice | Rationale |
|----------|--------|-----------|
| TUI framework | Bubble Tea v2 | Industry standard, Elm architecture, rich component library |
| Image lib | disintegration/imaging | Pure Go, no CGO, covers resize + JPEG encoding |
| Concurrency | errgroup + rate.Limiter | Bounded workers, backpressure, clean cancellation |
| API approach | REST API, not scraping | Stable, documented, no HTML parsing fragility |
| Multi-provider | Provider interface | Clean extensibility, add MangaKakalot without touching existing code |
| CBZ metadata | ComicInfo.xml | Standard format read by Komga, Kavita, Calibre, CDisplayEx |
| Dual mode | TUI default + CLI fallback | Interactive use + scripting/automation |

## API Reference (MangaDex v5)

```
Base: https://api.mangadex.org

GET /manga?title={query}&limit={n}&order[followedCount]=desc
  → Search manga by title. Returns list of manga with ID, title, description, tags, cover art.
  → Supports advanced filters: tags, demographic, status, year, content rating.

GET /manga/{id}
  → Manga title, description, tags, author, artist, status, availableTranslatedLanguages

GET /manga/{id}/feed?translatedLanguage[]={lang}&order[chapter]=asc&limit=100&offset={n}
  → Paginated chapter list (max 100/page)
  → Each chapter: id, volume, chapter number, title, pages, translatedLanguage

GET /at-home/server/{chapterId}
  → { baseUrl, chapter: { hash, data[], dataSaver[] } }
  → Full image: {baseUrl}/data/{hash}/{filename}
  → Compressed: {baseUrl}/data-saver/{hash}/{filename}

Cover art:
  GET /cover?manga[]={mangaId}&limit=100
  → Returns cover filenames. Full URL: https://uploads.mangadex.org/covers/{mangaId}/{fileName}

Rate limits:
  - General API: ~5 req/s
  - At-home CDN: ~1 req/s (be conservative)
```

## Memory Map

| File | Purpose |
|------|---------|
| `AGENTS.md` | Project context for AI agents (stack, structure, conventions) |
| `PLAN.md` | This file — implementation phases, decisions, API reference |
| `~/.comicdown.yaml` | Future: user config (default language, quality, output dir) |
