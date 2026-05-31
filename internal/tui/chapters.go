package tui

import (
	"fmt"
	"strings"

	"github.com/sasmitai/comicdown/internal/mangadex"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ChaptersModel is Screen 2: scrollable chapter list with checkboxes.
type ChaptersModel struct {
	manga    mangaInfo
	chapters []chapterInfo
	selected map[int]bool
	cursor   int
	offset   int
	visible  int
	filter   string
	filtered []int
	width    int
	height   int
}

func NewChaptersModel(manga mangaInfo, chapters []chapterInfo, width, height int) ChaptersModel {
	selected := make(map[int]bool, len(chapters))
	for i := range chapters {
		selected[i] = true // select all by default
	}
	m := ChaptersModel{
		manga:    manga,
		chapters: chapters,
		selected: selected,
		width:    width,
		height:   height,
	}
	m.calcVisible()
	m.refilter()
	return m
}

func (m *ChaptersModel) calcVisible() {
	// Total height minus header (5 lines) and footer (3 lines).
	m.visible = m.height - 8
	if m.visible < 5 {
		m.visible = 5
	}
}

func (m ChaptersModel) Init() tea.Cmd { return nil }

func (m ChaptersModel) Update(msg tea.Msg) (ChaptersModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.calcVisible()
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				if m.cursor < m.offset {
					m.offset = m.cursor
				}
			}
		case "down", "j":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
				if m.cursor >= m.offset+m.visible {
					m.offset = m.cursor - m.visible + 1
				}
			}
		case "space": // Space toggles selection
			if len(m.filtered) > 0 {
				idx := m.filtered[m.cursor]
				m.selected[idx] = !m.selected[idx]
			}
		case "enter":
			return m, func() tea.Msg { return downloadStartMsg{} }
		case "a":
			all := true
			for _, idx := range m.filtered {
				if !m.selected[idx] {
					all = false
					break
				}
			}
			for _, idx := range m.filtered {
				m.selected[idx] = !all
			}
		case "n":
			for _, idx := range m.filtered {
				m.selected[idx] = false
			}
		case "esc":
			return m, func() tea.Msg { return backToInputMsg{} }
		case "backspace":
			if len(m.filter) > 0 {
				m.filter = m.filter[:len(m.filter)-1]
				m.refilter()
			}
		default:
			if len(msg.String()) == 1 && msg.String()[0] >= 32 {
				m.filter += msg.String()
				m.refilter()
				m.cursor = 0
				m.offset = 0
			}
		}
	}
	return m, nil
}

func (m ChaptersModel) View(width int) string {
	w := width
	if w == 0 {
		w = 80
	}

	var b strings.Builder

	// ── Title bar ──
	title := lipgloss.NewStyle().
		Width(w).
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Align(lipgloss.Center).
		Padding(1, 0).
		Render(fmt.Sprintf("📖  %s", m.manga.Title))
	b.WriteString(title)
	b.WriteString("\n")

	// ── Info bar ──
	selCount := m.countSelected()
	lang := "N/A"
	if len(m.chapters) > 0 {
		lang = m.chapters[0].TranslatedLanguage
	}
	infoBar := lipgloss.NewStyle().
		Width(w).
		Foreground(lipgloss.Color("#A0A0A0")).
		Background(lipgloss.Color("#2D2D2D")).
		Padding(0, 2).
		Render(fmt.Sprintf("%d chapters total • %d selected • lang: %s", len(m.chapters), selCount, lang))
	b.WriteString(infoBar)
	b.WriteString("\n\n")

	// ── Filter ──
	if m.filter != "" {
		b.WriteString(fmt.Sprintf("    Filter: %s\n\n",
			lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Render(m.filter),
		))
	}

	// ── List ──
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FAFAFA")).Bold(true)
	unselectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
	checkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF87"))

	end := m.offset + m.visible
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	for pos := m.offset; pos < end; pos++ {
		idx := m.filtered[pos]
		ch := m.chapters[idx]

		cursor := "  "
		if pos == m.cursor {
			cursor = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Render("▶ ")
		}

		check := "☐"
		if m.selected[idx] {
			check = checkStyle.Render("☑")
		}

		vol := ""
		if ch.Volume != "" {
			vol = fmt.Sprintf("Vol.%s", ch.Volume)
		}

		// Fixed widths for columns
		line := fmt.Sprintf("%s %-7s %-8s %-45s %3dp",
			check,
			fmt.Sprintf("Ch.%s", ch.Chapter),
			vol,
			truncate(ch.Title, 45),
			ch.Pages,
		)

		if m.selected[idx] {
			b.WriteString(fmt.Sprintf("  %s%s\n", cursor, selectedStyle.Render(line)))
		} else {
			b.WriteString(fmt.Sprintf("  %s%s\n", cursor, unselectedStyle.Render(line)))
		}
	}

	// Pad with empty lines if needed to keep UI stable
	for i := len(m.filtered); i < m.visible; i++ {
		b.WriteString("\n")
	}

	// ── Footer ──
	b.WriteString("\n")
	help := lipgloss.NewStyle().
		Width(w).
		Foreground(lipgloss.Color("#626262")).
		Background(lipgloss.Color("#1A1A1A")).
		Padding(0, 2).
		Render("[a] all  [n] none  [/] type to filter  [Space] toggle  [Enter] download  [Esc] back")
	b.WriteString(help)

	return b.String()
}

// Selected returns chapters that are currently selected, in order.
func (m ChaptersModel) Selected() []chapterInfo {
	var out []chapterInfo
	for i, ch := range m.chapters {
		if m.selected[i] {
			out = append(out, ch)
		}
	}
	return out
}

func (m *ChaptersModel) refilter() {
	m.filtered = m.filtered[:0]
	if m.filter == "" {
		m.filtered = make([]int, len(m.chapters))
		for i := range m.chapters {
			m.filtered[i] = i
		}
		return
	}
	q := strings.ToLower(m.filter)
	for i, ch := range m.chapters {
		if strings.Contains(strings.ToLower(ch.Title), q) ||
			strings.Contains(ch.Chapter, q) {
			m.filtered = append(m.filtered, i)
		}
	}
}

func (m ChaptersModel) countSelected() int {
	n := 0
	for _, v := range m.selected {
		if v {
			n++
		}
	}
	return n
}

// fetchChaptersCmd fetches manga + chapter list from MangaDex and returns a message.
func fetchChaptersCmd(opts Opts) tea.Cmd {
	return func() tea.Msg {
		id, err := mangadex.ExtractMangaID(opts.URL)
		if err != nil {
			return chaptersLoadErrMsg{err: fmt.Errorf("invalid URL: %w", err)}
		}

		client := mangadex.NewClient()

		manga, err := client.FetchManga(id)
		if err != nil {
			return chaptersLoadErrMsg{err: fmt.Errorf("fetch manga: %w", err)}
		}

		chapters, err := client.FetchAllChapters(id, opts.Lang)
		if err != nil {
			return chaptersLoadErrMsg{err: fmt.Errorf("fetch chapters: %w", err)}
		}

		if len(chapters) == 0 {
			return chaptersLoadErrMsg{err: fmt.Errorf(
				"no %s chapters found for this manga", opts.Lang,
			)}
		}

		info := make([]chapterInfo, len(chapters))
		for i, ch := range chapters {
			info[i] = chapterInfo{
				ID:                 ch.ID,
				Volume:             ch.Volume,
				Chapter:            ch.Chapter,
				Title:              ch.Title,
				TranslatedLanguage: ch.TranslatedLanguage,
				Pages:              ch.Pages,
			}
		}

		return chaptersLoadedMsg{
			manga: mangaInfo{
				Title: manga.Title,
				Tags:  manga.Tags,
				Year:  manga.Year,
			},
			chapters: info,
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
