package tui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// SummaryModel is Screen 4.
type SummaryModel struct {
	states  []chapterState
	opts    Opts
	started time.Time
	width   int
	height  int
}

func NewSummaryModel(states []chapterState, opts Opts, started time.Time) SummaryModel {
	return SummaryModel{states: states, opts: opts, started: started}
}

func (m SummaryModel) Init() tea.Cmd { return nil }

func (m SummaryModel) Update(msg tea.Msg) (SummaryModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "o":
			openFolder(m.opts.Output)
			return m, nil
		case "d":
			return m, func() tea.Msg { return backToInputMsg{} }
		case "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m SummaryModel) View(width int) string {
	w := width
	if w == 0 {
		w = 80
	}

	var b strings.Builder

	doneCount, errorCount := 0, 0
	var totalSize int64
	var totalPages int
	for _, s := range m.states {
		if s.status == "done" {
			doneCount++
			totalSize += s.cbzSize
			totalPages += s.pagesTotal
		} else if s.status == "error" {
			errorCount++
		}
	}

	elapsed := time.Since(m.started).Round(time.Second)

	// ── Title ──
	title := lipgloss.NewStyle().
		Width(w).
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#00B55A")).
		Align(lipgloss.Center).
		Padding(1, 0).
		Render("✅  Download Complete!")
	b.WriteString(title)
	b.WriteString("\n\n")

	// ── Summary Box ──
	lblStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#B8B8B8")).Width(10)
	valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FAFAFA")).Bold(true)

	b.WriteString(fmt.Sprintf("    %s %s\n", lblStyle.Render("Chapters"), valStyle.Render(fmt.Sprintf("%d/%d downloaded", doneCount, len(m.states)))))
	if errorCount > 0 {
		b.WriteString(fmt.Sprintf("    %s %s\n", lblStyle.Render("Failed"), errorStyle.Render(fmt.Sprintf("%d", errorCount))))
	}
	b.WriteString(fmt.Sprintf("    %s %s\n", lblStyle.Render("Pages"), valStyle.Render(fmt.Sprintf("%d", totalPages))))
	b.WriteString(fmt.Sprintf("    %s %s\n", lblStyle.Render("Size"), valStyle.Render(formatSize(totalSize))))
	b.WriteString(fmt.Sprintf("    %s %s\n", lblStyle.Render("Time"), valStyle.Render(elapsed.String())))
	b.WriteString(fmt.Sprintf("    %s %s\n", lblStyle.Render("Output"), valStyle.Render(m.opts.Output)))
	b.WriteString("\n")

	// ── File list ──
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3C3C3C")).
		Padding(0, 1)

	var files strings.Builder
	rendered := 0
	maxFiles := 10
	if m.height > 30 {
		maxFiles = m.height - 18
	}

	for _, s := range m.states {
		if s.status != "done" {
			continue
		}
		if rendered >= maxFiles {
			files.WriteString(dimStyle.Render(fmt.Sprintf("    ... and %d more files\n", doneCount-rendered)))
			break
		}
		files.WriteString(fmt.Sprintf("📦 %-45s %s\n",
			truncate(filepath.Base(s.cbzPath), 45),
			dimStyle.Render(formatSize(s.cbzSize)),
		))
		rendered++
	}
	
	if doneCount > 0 {
		b.WriteString("    ")
		b.WriteString(boxStyle.Render(strings.TrimSuffix(files.String(), "\n")))
		b.WriteString("\n")
	}

	// ── Errors ──
	if errorCount > 0 {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render(fmt.Sprintf("    %d chapters failed:", errorCount)))
		b.WriteString("\n")
		for _, s := range m.states {
			if s.status == "error" {
				b.WriteString(dimStyle.Render(fmt.Sprintf("      Ch.%s %s: %s\n",
					s.chapter.Chapter, s.chapter.Title, s.errMsg)))
			}
		}
	}

	// ── Footer ──
	b.WriteString("\n")
	help := lipgloss.NewStyle().
		Width(w).
		Foreground(lipgloss.Color("#626262")).
		Background(lipgloss.Color("#1A1A1A")).
		Padding(0, 2).
		Render("[o] Open folder  [d] Download more  [q] Quit")
	b.WriteString(help)

	return b.String()
}

func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
	)
	switch {
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func openFolder(path string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "linux":
		cmd = exec.Command("xdg-open", path)
	case "windows":
		cmd = exec.Command("explorer", path)
	}
	if cmd != nil {
		cmd.Start()
	}
}
