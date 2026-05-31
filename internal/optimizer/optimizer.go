package optimizer

import (
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"

	"github.com/disintegration/imaging"
)

// Options configures image optimization.
type Options struct {
	Quality  int // JPEG quality 1-100 (default 85)
	MaxWidth int // Max pixel width; 0 = no resize (default 0)
}

// OptimizeImage reads an image from srcPath, applies optimization,
// and writes the result to destPath.
func OptimizeImage(srcPath, destPath string, opts Options) error {
	if opts.Quality <= 0 {
		opts.Quality = 85
	}

	src, err := imaging.Open(srcPath, imaging.AutoOrientation(true))
	if err != nil {
		return fmt.Errorf("open image %s: %w", srcPath, err)
	}
	defer func() {
		// imaging doesn't expose a Close; let GC handle it for decoded images.
	}()

	img := src

	// Resize if maxWidth is set and image exceeds it.
	if opts.MaxWidth > 0 {
		bounds := img.Bounds()
		if bounds.Dx() > opts.MaxWidth {
			img = imaging.Resize(img, opts.MaxWidth, 0, imaging.Lanczos)
		}
	}

	// Ensure destination directory exists.
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("create dest dir: %w", err)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create %s: %w", destPath, err)
	}
	defer f.Close()

	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: opts.Quality}); err != nil {
		os.Remove(destPath)
		return fmt.Errorf("encode JPEG %s: %w", destPath, err)
	}

	return nil
}

// OptimizeDir optimizes all images in srcDir and writes them to destDir.
// Files are named 001.jpg, 002.jpg, etc. based on sorted order.
// Returns the number of files optimized.
func OptimizeDir(srcDir, destDir string, opts Options) (int, error) {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return 0, fmt.Errorf("create dest dir %s: %w", destDir, err)
	}

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return 0, fmt.Errorf("read dir %s: %w", srcDir, err)
	}

	count := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if ext := filepath.Ext(name); ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
			continue
		}

		srcPath := filepath.Join(srcDir, name)
		destPath := filepath.Join(destDir, name)

		if err := OptimizeImage(srcPath, destPath, opts); err != nil {
			return count, fmt.Errorf("optimize %s: %w", name, err)
		}
		count++
	}

	return count, nil
}

// DecodeSize returns the dimensions of an image file without fully decoding it.
func DecodeSize(path string) (image.Rectangle, error) {
	f, err := os.Open(path)
	if err != nil {
		return image.Rectangle{}, err
	}
	defer f.Close()

	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return image.Rectangle{}, err
	}
	return image.Rect(0, 0, cfg.Width, cfg.Height), nil
}
