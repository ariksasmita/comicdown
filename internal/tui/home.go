package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type entryMode int

const (
	entryPasteURL entryMode = iota
	entrySearch
)

// HomeModel is Screen 0: choose between pasting a URL or searching.
type HomeModel struct {
	cursor int
	items  []homeItem
	width  int
	height int
}

type homeItem struct {
	icon        string
	label       string
	description string
}

func NewHomeModel() HomeModel {
	return HomeModel{
		items: []homeItem{
			{icon: "🔗", label: "Paste URL", description: "Paste a link from your browser"},
			{icon: "🔍", label: "Search by Title", description: "Search across manga providers"},
		},
	}
}

func (m HomeModel) Init() tea.Cmd { return nil }

func (m HomeModel) Update(msg tea.Msg) (HomeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter":
			return m, func() tea.Msg { return homeSelectMsg{mode: entryMode(m.cursor)} }
		}
	}
	return m, nil
}

func (m HomeModel) View() string {
	w := m.width
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
		Render("📚  ComicDown")
	b.WriteString(title)
	b.WriteString("\n\n")

	b.WriteString(lipgloss.NewStyle().Padding(0, 4).Render("How would you like to find a manga?"))
	b.WriteString("\n\n")

	for i, item := range m.items {
		cursor := "  "
		if i == m.cursor {
			cursor = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Render("▶ ")
		}

		labelStyle := lipgloss.NewStyle().Bold(true).Width(w - 16)
		descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))

		if i == m.cursor {
			cardStyle := lipgloss.NewStyle().
				Width(w - 8).
				Padding(1, 2).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#7D56F4"))
			card := fmt.Sprintf("%s %s\n   %s", item.icon, labelStyle.Render(item.label), descStyle.Render(item.description))
			b.WriteString(lipgloss.NewStyle().Padding(0, 3).Render(cursor + cardStyle.Render(card)))
		} else {
			cardStyle := lipgloss.NewStyle().
				Width(w - 8).
				Padding(1, 2).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#3C3C3C"))
			card := fmt.Sprintf("%s %s\n   %s", item.icon, labelStyle.Render(item.label), descStyle.Render(item.description))
			b.WriteString(lipgloss.NewStyle().Padding(0, 3).Render(cursor + cardStyle.Render(card)))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	help := lipgloss.NewStyle().
		Width(w).
		Foreground(lipgloss.Color("#626262")).
		Background(lipgloss.Color("#1A1A1A")).
		Padding(0, 2).
		Render("↑↓  navigate  •  Enter  select  •  Ctrl+C  quit")
	b.WriteString(help)

	return b.String()
}
