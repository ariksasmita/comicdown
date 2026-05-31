package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
)

const (
	userAgent      = "comicdown/1.0 (github.com/sasmitai/comicdown)"
	maxRetries     = 3
	retryBaseDelay = 500 // ms
)

// ProgressFn is called after each page download with (completed, total) counts.
type ProgressFn func(completed, total int)

// Options configures the download behaviour.
type Options struct {
	Workers   int // concurrent download goroutines (default 3)
	RateLimit int // requests per second for image CDN (default 1)
}

// DownloadPages downloads all page images from urls into destDir.
// Files are named 001.jpg, 002.jpg, etc. based on order.
// Calls onProgress after each successful download.
func DownloadPages(ctx context.Context, urls []string, destDir string, opts Options, onProgress ProgressFn) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("create dest dir %s: %w", destDir, err)
	}

	if opts.Workers <= 0 {
		opts.Workers = 3
	}
	if opts.RateLimit <= 0 {
		opts.RateLimit = 1
	}

	limiter := rate.NewLimiter(rate.Limit(opts.RateLimit), opts.RateLimit)
	client := &http.Client{Timeout: 60 * time.Second}

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(opts.Workers)

	var mu sync.Mutex
	completed := 0
	total := len(urls)

	for i, rawURL := range urls {
		i, rawURL := i, rawURL

		g.Go(func() error {
			// Respect rate limiter
			if err := limiter.Wait(gctx); err != nil {
				return fmt.Errorf("rate limiter: %w", err)
			}

			filename := filepath.Join(destDir, pageFilename(i))
			if err := downloadWithRetry(client, rawURL, filename); err != nil {
				return fmt.Errorf("page %d (%s): %w", i+1, rawURL, err)
			}

			mu.Lock()
			completed++
			if onProgress != nil {
				onProgress(completed, total)
			}
			mu.Unlock()

			return nil
		})
	}

	return g.Wait()
}

// downloadWithRetry attempts to download a file with exponential backoff.
func downloadWithRetry(client *http.Client, url, dest string) error {
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(retryBaseDelay*(1<<(attempt-1))) * time.Millisecond)
		}
		lastErr = downloadFile(client, url, dest)
		if lastErr == nil {
			return nil
		}
	}
	return lastErr
}

// downloadFile performs a single HTTP GET and writes the body to dest.
func downloadFile(client *http.Client, url, dest string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP GET: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		os.Remove(dest)
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// pageFilename returns a zero-padded filename like "001.jpg".
func pageFilename(index int) string {
	return fmt.Sprintf("%03d.jpg", index+1)
}
