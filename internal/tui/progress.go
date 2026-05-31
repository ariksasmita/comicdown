package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sasmitai/comicdown/internal/downloader"
	"github.com/sasmitai/comicdown/internal/mangadex"
	"github.com/sasmitai/comicdown/internal/optimizer"
	"github.com/sasmitai/comicdown/internal/packager"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type chapterState struct {
	chapter    chapterInfo
	status     string // pending | downloading | optimizing | packaging | done | error
	progress   float64
	pagesDone  int
	pagesTotal int
	cbzPath    string
	cbzSize    int64
	errMsg     string
}

// ProgressModel is Screen 3.
type ProgressModel struct {
	states  []chapterState
	opts    Opts
	width   int
	height  int
	started time.Time
	overall float64
	manga   mangaInfo
}

// Messages
type chapterDoneMsg struct {
	index   int
	cbzPath string
	cbzSize int64
	err     error
}
type allDoneMsg struct{}

func NewProgressModel(chapters []chapterInfo, opts Opts, width, height int, manga mangaInfo) ProgressModel {
	states := make([]chapterState, len(chapters))
	for i, ch := range chapters {
		states[i] = chapterState{
			chapter:    ch,
			status:     "pending",
			pagesTotal: ch.Pages,
		}
	}
	return ProgressModel{
		states:  states,
		opts:    opts,
		width:   width,
		height:  height,
		manga:   manga,
		started: time.Now(),
	}
}

func (m ProgressModel) Init() tea.Cmd {
	if len(m.states) == 0 {
		return func() tea.Msg { return allDoneMsg{} }
	}
	return m.downloadChapter(0)
}

func (m ProgressModel) Update(msg tea.Msg) (ProgressModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case chapterDoneMsg:
		if msg.err != nil {
			m.states[msg.index].status = "error"
			m.states[msg.index].errMsg = msg.err.Error()
		} else {
			m.states[msg.index].status = "done"
			m.states[msg.index].progress = 1.0
			m.states[msg.index].pagesDone = m.states[msg.index].pagesTotal
			m.states[msg.index].cbzPath = msg.cbzPath
			m.states[msg.index].cbzSize = msg.cbzSize
		}
		m.recalcOverall()

		next := msg.index + 1
		if next < len(m.states) {
			m.states[next].status = "downloading"
			return m, m.downloadChapter(next)
		}
		return m, func() tea.Msg { return allDoneMsg{} }

	case allDoneMsg:
		return m, func() tea.Msg {
			return downloadDoneMsg{summary: NewSummaryModel(m.states, m.opts, m.started)}
		}

	case tea.KeyPressMsg:
		if msg.String() == "esc" || msg.String() == "q" {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m ProgressModel) View(width int) string {
	w := width
	if w == 0 {
		w = 80
	}

	var b strings.Builder

	done := 0
	for _, s := range m.states {
		if s.status == "done" {
			done++
		}
	}

	// ── Title ──
	title := lipgloss.NewStyle().
		Width(w).
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Align(lipgloss.Center).
		Padding(1, 0).
		Render(fmt.Sprintf("📥  Downloading — %d/%d chapters", done, len(m.states)))
	b.WriteString(title)
	b.WriteString("\n\n")

	// ── Items ──
	barWidth := 20
	maxRows := 15
	if m.height > 25 {
		maxRows = m.height - 10
	}

	for i, s := range m.states {
		if i >= maxRows && i < len(m.states)-1 {
			remaining := len(m.states) - maxRows
			b.WriteString(dimStyle.Render(fmt.Sprintf("    … and %d more chapters\n", remaining)))
			break
		}

		icon := statusIcon(s.status)
		bar := progressBar(s.progress, barWidth)
		label := fmt.Sprintf("Ch.%-4s %-25s %dp",
			s.chapter.Chapter,
			truncate(s.chapter.Title, 25),
			s.chapter.Pages,
		)
		row := fmt.Sprintf("    %s %s %s", icon, bar, label)
		if s.status == "error" {
			b.WriteString(errorStyle.Render(row))
			b.WriteString("  " + dimStyle.Render(truncate(s.errMsg, 30)))
		} else {
			b.WriteString(row)
		}
		b.WriteString("\n")
	}

	// ── Overall ──
	b.WriteString("\n")
	elapsed := time.Since(m.started).Round(time.Second)
	
	overallBar := progressBar(m.overall, barWidth+10)
	statStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FAFAFA")).Bold(true)
	
	b.WriteString(fmt.Sprintf("    %s  %s\n",
		statStyle.Render("Overall"),
		overallBar,
	))
	b.WriteString(fmt.Sprintf("    %s\n",
		dimStyle.Render(fmt.Sprintf("%d/%d chapters • elapsed: %s", done, len(m.states), elapsed.String())),
	))
	
	// ── Footer ──
	b.WriteString("\n")
	help := lipgloss.NewStyle().
		Width(w).
		Foreground(lipgloss.Color("#626262")).
		Background(lipgloss.Color("#1A1A1A")).
		Padding(0, 2).
		Render("[Esc/q] abort")
	b.WriteString(help)

	return b.String()
}

func (m *ProgressModel) recalcOverall() {
	var totalPages, donePages int
	for _, s := range m.states {
		totalPages += s.pagesTotal
		donePages += s.pagesDone
	}
	if totalPages > 0 {
		m.overall = float64(donePages) / float64(totalPages)
	}
}

func (m ProgressModel) downloadChapter(index int) tea.Cmd {
	return func() tea.Msg {
		ch := m.states[index].chapter
		client := mangadex.NewClient()

		pageURLs, err := client.FetchPageURLs(ch.ID)
		if err != nil {
			return chapterDoneMsg{index: index, err: fmt.Errorf("fetch page URLs: %w", err)}
		}

		urls := pageURLs.Original
		if m.opts.DataSaver {
			urls = pageURLs.Saver
		}

		tmpDir := filepath.Join(os.TempDir(), "comicdown", ch.ID)
		defer os.RemoveAll(tmpDir)

		err = downloader.DownloadPages(context.Background(), urls, tmpDir,
			downloader.Options{Workers: m.opts.Workers, RateLimit: 1},
			nil,
		)
		if err != nil {
			return chapterDoneMsg{index: index, err: fmt.Errorf("download: %w", err)}
		}

		optDir := filepath.Join(os.TempDir(), "comicdown", ch.ID+"_opt")
		defer os.RemoveAll(optDir)

		_, err = optimizer.OptimizeDir(tmpDir, optDir, optimizer.Options{
			Quality:  m.opts.Quality,
			MaxWidth: m.opts.MaxWidth,
		})
		if err != nil {
			return chapterDoneMsg{index: index, err: fmt.Errorf("optimize: %w", err)}
		}

		if err := os.MkdirAll(m.opts.Output, 0o755); err != nil {
			return chapterDoneMsg{index: index, err: fmt.Errorf("create output dir: %w", err)}
		}

		seriesTitle := m.manga.Title
		if seriesTitle == "" {
			seriesTitle = ch.Title
		}

		cbzName := packager.ChapterFileName(seriesTitle, ch.Chapter, ch.Title)
		cbzPath := filepath.Join(m.opts.Output, cbzName)

		err = packager.CreateCBZ(optDir, cbzPath, packager.ComicInfo{
			Series:      seriesTitle,
			Number:      ch.Chapter,
			Volume:      ch.Volume,
			Title:       ch.Title,
			LanguageISO: ch.TranslatedLanguage,
			Manga:       "Yes",
		})
		if err != nil {
			return chapterDoneMsg{index: index, err: fmt.Errorf("package CBZ: %w", err)}
		}

		fi, _ := os.Stat(cbzPath)
		var size int64
		if fi != nil {
			size = fi.Size()
		}
		return chapterDoneMsg{index: index, cbzPath: cbzPath, cbzSize: size}
	}
}

func statusIcon(status string) string {
	switch status {
	case "done":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF87")).Render("✓")
	case "downloading", "optimizing", "packaging":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FFCC00")).Render("↻")
	case "error":
		return errorStyle.Render("✗")
	default:
		return dimStyle.Render("○")
	}
}

func progressBar(pct float64, width int) string {
	filled := int(pct * float64(width))
	if filled > width {
		filled = width
	}
	filledStr := strings.Repeat("█", filled)
	emptyStr := strings.Repeat("░", width-filled)
	pctStr := fmt.Sprintf("%3.0f%%", pct*100)
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Render(filledStr) +
		dimStyle.Render(emptyStr) + " " + pctStr
}
