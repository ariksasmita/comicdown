# 📚 ComicDown

A terminal-based manga downloader for [MangaDex](https://mangadex.org). Downloads chapters, optimizes images, and packages them as CBZ archives ready for your favorite comic reader (Komga, Kavita, Calibre, CDisplayEx, etc.).

Built with Go + [Bubble Tea v2](https://charm.land/blog/v2/) for the TUI.

## Features

- **Interactive TUI** — Browse and select chapters with keyboard navigation
- **Search by title** — Find manga directly inside the TUI, no browser needed
- **Multiple providers** — MangaDex now, MangaKakalot coming soon (auto-detected from URL)
- **CLI headless mode** — Scriptable for automation and batch downloads
- **Image optimization** — Resize and compress JPEGs with configurable quality
- **CBZ packaging** — Standard ZIP with [ComicInfo.xml](https://github.com/anansi-project/comicinfo) metadata
- **HTTP file server** — Mobile-friendly web UI to download CBZ files from other devices
- **File fixer** — Rename and fix metadata in already-downloaded files
- **Paste support** — Ctrl+V works in all TUI input fields

## Install

```bash
git clone https://github.com/ariksasmita/comicdown.git
cd comicdown
go build ./cmd/comicdown/
```

## Usage

### Interactive TUI

```bash
./comicdown
```

Navigates through 6 screens:
1. **Home** — Choose between pasting a URL or searching by title
2. **Search** — Search manga by title across providers (TUI only)
3. **Input** — Paste a MangaDex URL, set language/quality/output
4. **Chapters** — Select which chapters to download (space to toggle, `a` select all, `n` deselect all, type to filter)
5. **Progress** — Live download/optimize/package progress with per-chapter bars
6. **Summary** — Results with file sizes, open folder option

### CLI / Headless

```bash
# Download specific chapters
./comicdown --url "https://mangadex.org/title/..." --lang en --from 1 --to 10

# Download all chapters
./comicdown --url "https://mangadex.org/title/..." --lang en

# All options
./comicdown --url <URL> [options]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--url` | — | MangaDex manga URL (required for CLI mode) |
| `--lang` | `en` | Language code |
| `--from` | — | Start chapter number (inclusive) |
| `--to` | — | End chapter number (inclusive) |
| `--quality` | `85` | JPEG quality (1–100) |
| `--max-width` | `1600` | Max image width in px, `0` = no resize |
| `--output` | `./output` | Output directory |
| `--data-saver` | `false` | Use MangaDex compressed source images |
| `--workers` | `3` | Concurrent download workers |

### File Server

Share your downloads with other devices on the same network:

```bash
./comicdown --serve --output ./output
```

Opens a mobile-friendly web UI at `http://<your-ip>:18080` with:
- Large tap-friendly cards
- Files grouped by download date
- Direct download links

| Flag | Default | Description |
|------|---------|-------------|
| `--serve` | — | Start HTTP file server |
| `--port` | `18080` | Port to listen on |
| `--output` | `./output` | Directory to serve |

### Fix File Names

Rename existing CBZ files to the correct format and fix ComicInfo.xml metadata:

```bash
./comicdown --fix-names --url "https://mangadex.org/title/..." --output ./output
```

This fetches the correct manga title and chapter list from MangaDex, then renames files from:
```
Chapter Title - Ch.1 Chapter Title.cbz
```
to:
```
Manga Title - Ch.1 Chapter Title.cbz
```

## Architecture

```
internal/
  provider/      # Provider interface + shared types (multi-source support)
  mangadex/      # MangaDex API client (implements Provider)
  downloader/    # Concurrent image downloader with rate limiting + retry
  optimizer/     # JPEG resize + compress (disintegration/imaging)
  packager/      # CBZ (ZIP) + ComicInfo.xml generation
  tui/           # Bubble Tea v2 screens (home, search, results, input, chapters, progress, summary)
```

## Data Source

Uses the [MangaDex REST API v5](https://api.mangadex.org) — no scraping, no auth required.

| Endpoint | Purpose |
|----------|---------|
| `GET /manga/{id}` | Manga metadata |
| `GET /manga/{id}/feed` | Chapter list (paginated, filterable by language) |
| `GET /at-home/server/{id}` | Image CDN URLs |

Rate limits: ~5 req/s general API, ~1 req/s image CDN.

## Output Format

Each chapter is packaged as a standard CBZ file:

```
Manga Title - Ch.1 Chapter Title.cbz
├── ComicInfo.xml   # Metadata (Series, Title, Number, Volume, Language, etc.)
├── 001.jpg
├── 002.jpg
└── ...
```

## Tech Stack

- **Language**: Go 1.26
- **TUI**: [Bubble Tea v2](https://charm.land/bubbletea/v2/) + [Lip Gloss v2](https://charm.land/lipgloss/v2/)
- **Image processing**: [disintegration/imaging](https://github.com/disintegration/imaging)
- **Rate limiting**: [golang.org/x/time/rate](https://pkg.go.dev/golang.org/x/time/rate)
- **Concurrency**: [golang.org/x/sync/errgroup](https://pkg.go.dev/golang.org/x/sync/errgroup)

## License

MIT
