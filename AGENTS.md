# Project: comicdown

A terminal-based tool to download manga chapters from MangaDex, optimize images, and package them as CBZ archives. Built with Go and a rich TUI powered by Bubble Tea v2.

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
  - `GET /manga/{id}` — manga metadata
  - `GET /manga/{id}/feed` — chapter list (paginated, filterable by language)
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
│       └── main.go                # CLI entrypoint (flag parsing → TUI or headless)
├── internal/
│   ├── mangadex/
│   │   ├── client.go              # API client: manga, feed, at-home URLs
│   │   └── types.go               # API request/response structs
│   ├── downloader/
│   │   └── downloader.go          # Rate-limited concurrent image downloader
│   ├── optimizer/
│   │   └── optimizer.go           # Image optimization (resize, compress, strip EXIF)
│   ├── packager/
│   │   └── cbz.go                 # CBZ (ZIP) packaging + ComicInfo.xml generation
│   └── tui/
│       ├── app.go                 # Root model: screen routing, tea.Model impl
│       ├── input.go               # Screen 1: URL + settings input form
│       ├── chapters.go            # Screen 2: chapter selector (checkbox list)
│       ├── progress.go            # Screen 3: download/optimize/package progress
│       └── summary.go             # Screen 4: results summary + open folder
├── go.mod
└── go.sum
```

## Conventions

- **Go module name**: `github.com/user/comicdown` (update after first `go mod init`)
- **Error handling**: explicit error returns, no panics in library code. Wrap errors with context using `fmt.Errorf("...: %w", err)`.
- **Naming**: standard Go conventions — `camelCase` for unexported, `PascalCase` for exported. Acronyms uppercase (`CBZ`, `API`, `URL`).
- **Concurrency**: use `errgroup.Group` for bounded concurrent downloads. Rate limiter via `golang.org/x/time/rate`.
- **TUI pattern**: each screen is a struct with `Init()`, `Update()`, `View()` methods. Parent `app.go` delegates to active screen.
- **Async work**: `tea.Cmd` functions perform I/O off the main goroutine, send typed `tea.Msg` back to update the view.
- **No external dependencies in TUI**: all styling via lipgloss, all components via bubbles. No custom ANSI escapes.
- **Logging**: use `log` to stderr or file when `DEBUG=1` is set. Never log to stdout (occupied by TUI).

## Key Commands

- `go build ./cmd/comicdown` — Build the binary
- `go run ./cmd/comicdown` — Run in TUI mode (interactive)
- `go run ./cmd/comicdown --url <URL> --lang en` — Run in headless/CLI mode
- `go test ./...` — Run all tests
- `go test ./internal/mangadex/...` — Run mangadex client tests only
- `DEBUG=1 go run ./cmd/comicdown` — Run with debug logging to `debug.log`

## Architecture Notes

- **Pipeline**: Fetch manga → list chapters → select → download pages → optimize → package CBZ
- **MangaDex API**: Public, no auth required. Rate-limited. CDN base URL rotates per request.
- **CBZ format**: Standard ZIP with sequential page images (`001.jpg`, `002.jpg`, ...) + `ComicInfo.xml` metadata.
- **Image optimization**: Strip EXIF, resize to max dimension, re-encode as JPEG at configurable quality.
- **Dual mode**: Interactive TUI by default; headless CLI when `--url` flag is passed for scripting/automation.
- **Screen flow**: Input → Chapters → Progress → Summary. Each screen handles its own state and keybindings.
