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

## Phase 9: Polish & Testing

- [ ] Unit tests for mangadex client (mock HTTP)
- [ ] Unit tests for optimizer (test images)
- [ ] Unit tests for CBZ packaging
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
| CBZ metadata | ComicInfo.xml | Standard format read by Komga, Kavita, Calibre, CDisplayEx |
| Dual mode | TUI default + CLI fallback | Interactive use + scripting/automation |

## API Reference (MangaDex v5)

```
Base: https://api.mangadex.org

GET /manga/{id}
  → Manga title, description, tags, author, artist, status, availableTranslatedLanguages

GET /manga/{id}/feed?translatedLanguage[]={lang}&order[chapter]=asc&limit=100&offset={n}
  → Paginated chapter list (max 100/page)
  → Each chapter: id, volume, chapter number, title, pages, translatedLanguage

GET /at-home/server/{chapterId}
  → { baseUrl, chapter: { hash, data[], dataSaver[] } }
  → Full image: {baseUrl}/data/{hash}/{filename}
  → Compressed: {baseUrl}/data-saver/{hash}/{filename}

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
