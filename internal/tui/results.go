package tui

import (
	"fmt"
	"strings"

	"github.com/sasmitai/comicdown/internal/provider"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ResultsModel is Screen 0b: search results list.
type ResultsModel struct {
	results  []provider.SearchResult
	query    string
	provider string
	cursor   int
	offset   int
	visible  int
	width    int
	height   int
	filter   string
	filtered []int
}

func NewResultsModel(results []provider.SearchResult, query, prov string, width, height int) ResultsModel {
	m := ResultsModel{
		results:  results,
		query:    query,
		provider: prov,
		width:    width,
		height:   height,
	}
	m.calcVisible()
	m.refilter()
	return m
}

func (m *ResultsModel) calcVisible() {
	m.visible = m.height - 8
	if m.visible < 5 {
		m.visible = 5
	}
}

func (m ResultsModel) Init() tea.Cmd { return nil }

func (m ResultsModel) Update(msg tea.Msg) (ResultsModel, tea.Cmd) {
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
		case "enter":
			if len(m.filtered) > 0 {
				idx := m.filtered[m.cursor]
				return m, func() tea.Msg {
					return resultSelectMsg{result: m.results[idx]}
				}
			}
		case "esc":
			return m, func() tea.Msg { return backToSearchMsg{} }
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

func (m ResultsModel) View(width int) string {
	w := width
	if w == 0 {
		w = 80
	}

	var b strings.Builder

	title := lipgloss.NewStyle().
		Width(w).
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Align(lipgloss.Center).
		Padding(1, 0).
		Render(fmt.Sprintf("🔍  \"%s\" — %s — %d results", m.query, m.provider, len(m.filtered)))
	b.WriteString(title)
	b.WriteString("\n")

	if m.filter != "" {
		b.WriteString(fmt.Sprintf("\n  Filter: %s\n",
			lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Render(m.filter),
		))
	}
	b.WriteString("\n")

	end := m.offset + m.visible
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	for pos := m.offset; pos < end; pos++ {
		idx := m.filtered[pos]
		r := m.results[idx]

		cursor := "  "
		if pos == m.cursor {
			cursor = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Render("▶ ")
		}

		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA"))
		statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
		tagStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))

		b.WriteString(fmt.Sprintf("  %s%s\n", cursor, titleStyle.Render(r.Title)))

		meta := []string{}
		if r.Status != "" {
			meta = append(meta, r.Status)
		}
		if r.Year > 0 {
			meta = append(meta, fmt.Sprintf("%d", r.Year))
		}
		if len(meta) > 0 {
			b.WriteString(fmt.Sprintf("     %s\n", statusStyle.Render(strings.Join(meta, " • "))))
		}

		if len(r.Tags) > 0 {
			tags := r.Tags
			if len(tags) > 4 {
				tags = tags[:4]
			}
			b.WriteString(fmt.Sprintf("     %s\n", tagStyle.Render(strings.Join(tags, ", "))))
		}

		if r.Description != "" {
			desc := r.Description
			if len(desc) > 80 {
				desc = desc[:77] + "..."
			}
			b.WriteString(fmt.Sprintf("     %s\n", statusStyle.Render(desc)))
		}

		b.WriteString("\n")
	}

	if len(m.filtered) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Padding(0, 4).Render("No results found."))
		b.WriteString("\n")
	}

	help := lipgloss.NewStyle().
		Width(w).
		Foreground(lipgloss.Color("#626262")).
		Background(lipgloss.Color("#1A1A1A")).
		Padding(0, 2).
		Render("↑↓ navigate • Enter select • type to filter • Esc back • Ctrl+C quit")
	b.WriteString(help)

	return b.String()
}

func (m *ResultsModel) refilter() {
	m.filtered = m.filtered[:0]
	if m.filter == "" {
		m.filtered = make([]int, len(m.results))
		for i := range m.results {
			m.filtered[i] = i
		}
		return
	}
	q := strings.ToLower(m.filter)
	for i, r := range m.results {
		if strings.Contains(strings.ToLower(r.Title), q) {
			m.filtered = append(m.filtered, i)
		}
	}
}
