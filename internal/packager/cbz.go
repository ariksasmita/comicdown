package packager

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ComicInfo holds metadata for the ComicInfo.xml file embedded in CBZ archives.
type ComicInfo struct {
	XMLName     xml.Name `xml:"ComicInfo"`
	XMLVersion  string   `xml:"version,attr"`
	Title       string   `xml:"Title,omitempty"`
	Series      string   `xml:"Series"`
	Number      string   `xml:"Number,omitempty"`
	Volume      string   `xml:"Volume,omitempty"`
	PageCount   int      `xml:"PageCount"`
	LanguageISO string   `xml:"LanguageISO,omitempty"`
	Writer      string   `xml:"Writer,omitempty"`
	Genre       string   `xml:"Genre,omitempty"`
	Manga       string   `xml:"Manga"` // "Yes" or "No"
	Summary     string   `xml:"Summary,omitempty"`
	Year        int      `xml:"Year,omitempty"`
}

// CreateCBZ creates a CBZ (ZIP) file at destPath containing all images from srcDir
// and the given ComicInfo metadata.
// Images are included in sorted filename order.
func CreateCBZ(srcDir, destPath string, info ComicInfo) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("create CBZ parent dir: %w", err)
	}

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("read source dir %s: %w", srcDir, err)
	}

	// Collect and sort image files.
	var images []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".webp" {
			images = append(images, e.Name())
		}
	}
	sort.Strings(images)

	// Update page count in metadata.
	info.PageCount = len(images)
	info.XMLVersion = "1.0"

	// Generate ComicInfo.xml.
	comicInfoXML, err := xml.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal ComicInfo.xml: %w", err)
	}
	xmlContent := append([]byte(xml.Header), comicInfoXML...)

	// Create the ZIP file.
	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create CBZ file %s: %w", destPath, err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	// Add ComicInfo.xml.
	ciw, err := w.Create("ComicInfo.xml")
	if err != nil {
		return fmt.Errorf("create ComicInfo.xml in ZIP: %w", err)
	}
	if _, err := ciw.Write(xmlContent); err != nil {
		return fmt.Errorf("write ComicInfo.xml: %w", err)
	}

	// Add image files.
	for _, name := range images {
		srcPath := filepath.Join(srcDir, name)
		if err := addFileToZip(w, srcPath, name); err != nil {
			return fmt.Errorf("add %s to CBZ: %w", name, err)
		}
	}

	return nil
}

// addFileToZip adds a single file to the zip writer with the given archive name.
func addFileToZip(w *zip.Writer, srcPath, arcName string) error {
	sf, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer sf.Close()

	fi, err := sf.Stat()
	if err != nil {
		return err
	}

	fh, err := zip.FileInfoHeader(fi)
	if err != nil {
		return err
	}
	fh.Name = arcName
	fh.Method = zip.Store // CBZ readers expect stored (uncompressed) images

	aw, err := w.CreateHeader(fh)
	if err != nil {
		return err
	}

	_, err = io.Copy(aw, sf)
	return err
}

// ChapterFileName generates a CBZ filename for a chapter.
// Format: "{series} - Ch.{num} {title}.cbz"
func ChapterFileName(series, chNum, title string) string {
	name := fmt.Sprintf("%s - Ch.%s", sanitize(series), sanitize(chNum))
	if title != "" {
		name += " " + sanitize(title)
	}
	return name + ".cbz"
}

// sanitize removes characters that are invalid in filenames.
func sanitize(s string) string {
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "",
		"?", "",
		`"`, "'",
		"<", "",
		">", "",
		"|", "",
	)
	return replacer.Replace(s)
}
