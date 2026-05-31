# Project: comicdown

A terminal-based tool to download manga chapters from MangaDex (and future: MangaKakalot), optimize images, and package them as CBZ archives. Built with Go and a rich TUI powered by Bubble Tea v2. Supports search-by-title or paste-URL entry.

## Tech Stack

### Language
- Go 1.26 (darwin/arm64)

### TUI Framework
- Bubble Tea v2 (`charm.land/bubbletea/v2`) — Elm Architecture (Model/Update/View)
- Bubbles v2 (`charm.land/bubbles/v2`) — spinner, progress, textinput, viewport
- Lip Gloss v2 (`charm.land/lipgloss/v2`) — styling, borders, layout

### Core Libraries
- HTTP client: `net/http` with `golang.org/x/time/rate` for rate limiting
- Image processing: `github.com/disintegration/imaging` (resize, JPEG encode)
- CBZ packaging: `archive/zip` + `encoding/xml` (stdlib)
- CLI fallback: `flag` stdlib (non-interactive / scripting mode)

### Data Source
- MangaDex REST API v5.1 (`api.mangadex.org`)
  - `GET /manga?title={query}` — search manga by title
  - `GET /manga/{id}` — manga metadata
  - `GET /manga/{id}/feed` — chapter list (paginated, filterable by language)
  - `GET /cover?manga[]={id}` — cover art filenames
  - `GET /at-home/server/{chapterId}` — image CDN URLs (hash + filenames)
  - Image URL pattern: `{baseUrl}/data/{hash}/{filename}`
  - Rate limit: <1 req/s on at-home endpoint, ~5 req/s on general API
  - Two quality tiers: `data[]` (original) and `dataSaver[]` (compressed)

## Project Structure

```
comicdown/
├── AGENTS.md                      # This file — project context
├── PLAN.md                        # Implementation plan & phase tracking
├── cmd/
│   └── comicdown/
│       ├── main.go                # CLI entrypoint (flag parsing → TUI or headless)
│       ├── server.go              # HTTP file server with mobile UI
│       └── fix.go                 # --fix-names rename utility
├── internal/
│   ├── provider/                  # Provider interface + shared types
│   │   ├── provider.go            # Provider interface, SearchResult, SearchOpts
│   │   └── types.go               # Manga, Chapter, PageURLs (shared across providers)
│   ├── mangadex/
│   │   ├── client.go              # MangaDex API client (implements Provider)
│   │   └── types.go               # MangaDex-specific API response structs
│   ├── downloader/
│   │   └── downloader.go          # Rate-limited concurrent image downloader
│   ├── optimizer/
│   │   └── optimizer.go           # Image optimization (resize, compress, strip EXIF)
│   ├── packager/
│   │   └── cbz.go                 # CBZ (ZIP) packaging + ComicInfo.xml generation
│   └── tui/
│       ├── app.go                 # Root model: screen routing, tea.Model impl
│       ├── home.go                # Screen 0: home (paste URL vs search)
│       ├── search.go              # Screen 0a: search form (provider + title)
│       ├── results.go             # Screen 0b: search results list
│       ├── input.go               # Screen 1: URL + settings input form
│       ├── chapters.go            # Screen 2: chapter selector (checkbox list)
│       ├── progress.go            # Screen 3: download/optimize/package progress
│       └── summary.go             # Screen 4: results summary + open folder
├── go.mod
└── go.sum
```

## Conventions

- **Go module name**: `github.com/sasmitai/comicdown`
- **Error handling**: explicit error returns, no panics in library code. Wrap errors with context using `fmt.Errorf("...: %w", err)`.
- **Naming**: standard Go conventions — `camelCase` for unexported, `PascalCase` for exported. Acronyms uppercase (`CBZ`, `API`, `URL`).
- **Provider pattern**: all manga sources implement `provider.Provider` interface. TUI/CLI never import `mangadex` directly — only `provider`.
- **Concurrency**: use `errgroup.Group` for bounded concurrent downloads. Rate limiter via `golang.org/x/time/rate`.
- **TUI pattern**: each screen is a struct with `Init()`, `Update()`, `View()` methods. Parent `app.go` delegates to active screen.
- **Async work**: `tea.Cmd` functions perform I/O off the main goroutine, send typed `tea.Msg` back to update the view.
- **No external dependencies in TUI**: all styling via lipgloss, all components via bubbles. No custom ANSI escapes.
- **Logging**: use `log` to stderr or file when `DEBUG=1` is set. Never log to stdout (occupied by TUI).
- **Cover art**: MangaDex covers at `https://uploads.mangadex.org/covers/{mangaId}/{fileName}`. Fetch cover filename from `/cover?manga[]={id}` endpoint.

## Key Commands

- `go build ./cmd/comicdown` — Build the binary
- `go run ./cmd/comicdown` — Run in TUI mode (interactive)
- `go run ./cmd/comicdown --url <URL> --lang en` — Run in headless/CLI mode
- `go run ./cmd/comicdown --search "One Piece" --provider mangadex` — Search and download
- `go test ./...` — Run all tests
- `go test ./internal/provider/...` — Run provider tests only
- `go test ./internal/mangadex/...` — Run mangadex client tests only
- `DEBUG=1 go run ./cmd/comicdown` — Run with debug logging to `debug.log`

## Architecture Notes

- **Pipeline**: Fetch manga → list chapters → select → download pages → optimize → package CBZ
- **Provider interface**: `internal/provider/provider.go` defines `Provider` with methods for search, fetch manga, chapters, and page URLs. Each source (MangaDex, future: MangaKakalot) implements this interface.
- **MangaDex API**: Public, no auth required. Rate-limited. CDN base URL rotates per request.
- **Cover art**: MangaDex serves covers via `uploads.mangadex.org`. Cover filename fetched from `/cover` endpoint.
- **CBZ format**: Standard ZIP with sequential page images (`001.jpg`, `002.jpg`, ...) + `ComicInfo.xml` metadata.
- **Image optimization**: Strip EXIF, resize to max dimension, re-encode as JPEG at configurable quality.
- **Dual mode**: Interactive TUI by default; headless CLI when `--url` or `--search` flag is passed.
- **Screen flow (TUI)**: Home → Input/Search → Chapters → Progress → Summary. Home screen lets user pick between pasting a URL or searching by title.
- **Auto-detection**: If a pasted URL matches a known provider (mangadex.org, future: mangakakalot.gg), the provider is auto-selected.
