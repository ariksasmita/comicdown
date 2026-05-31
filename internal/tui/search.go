package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// SearchModel is Screen 0a: search form with provider dropdown.
type SearchModel struct {
	focusIndex int
	fields     []searchField
	provider   int
	providers  []string
	width      int
	height     int
}

type searchField struct {
	label       string
	value       string
	placeholder string
}

func NewSearchModel() SearchModel {
	return SearchModel{
		providers: []string{"MangaDex"},
		fields: []searchField{
			{label: "Title", placeholder: "e.g. One Piece"},
			{label: "Language", value: "en", placeholder: "en"},
		},
	}
}

func (m SearchModel) Init() tea.Cmd { return nil }

func (m SearchModel) Update(msg tea.Msg) (SearchModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "tab", "down":
			m.focusIndex = (m.focusIndex + 1) % (len(m.fields) + 1) // +1 for provider
			return m, nil
		case "shift+tab", "up":
			m.focusIndex = (m.focusIndex - 1 + len(m.fields) + 1) % (len(m.fields) + 1)
			return m, nil
		case "enter":
			return m, m.submit()
		case "backspace":
			if m.focusIndex < len(m.fields) {
				f := &m.fields[m.focusIndex]
				if len(f.value) > 0 {
					f.value = f.value[:len(f.value)-1]
				}
			}
			return m, nil
		case "esc":
			return m, func() tea.Msg { return backToHomeMsg{} }
		default:
			if len(msg.String()) == 1 && msg.String()[0] >= 32 {
				if m.focusIndex < len(m.fields) {
					m.fields[m.focusIndex].value += msg.String()
				}
			}
			return m, nil
		}

	case tea.PasteMsg:
		if m.focusIndex < len(m.fields) {
			pasted := strings.ReplaceAll(msg.Content, "\n", "")
			m.fields[m.focusIndex].value += pasted
		}
		return m, nil
	}

	return m, nil
}

func (m SearchModel) View() string {
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
		Render("🔍  Search Manga")
	b.WriteString(title)
	b.WriteString("\n\n")

	formWidth := min(w-8, 70)
	focusedBorder := lipgloss.NewStyle().Width(formWidth).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#7D56F4"))
	normalBorder := lipgloss.NewStyle().Width(formWidth).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#3C3C3C"))

	// Provider selector
	border := normalBorder
	if m.focusIndex == len(m.fields) {
		border = focusedBorder
	}
	b.WriteString("    Provider\n")
	provValue := m.providers[m.provider]
	b.WriteString("    ")
	b.WriteString(border.Render("  " + provValue + "  "))
	b.WriteString("\n\n")

	// Text fields
	for i, f := range m.fields {
		border := normalBorder
		if i == m.focusIndex {
			border = focusedBorder
		}

		value := f.value
		if value == "" {
			value = lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).Render(f.placeholder)
		}

		b.WriteString("    " + f.label + "\n")
		b.WriteString("    ")
		b.WriteString(border.Render("  " + value + "  "))
		b.WriteString("\n\n")
	}

	// Search button
	btn := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Bold(true).
		Padding(0, 3).
		Render("Search")
	b.WriteString(lipgloss.NewStyle().Width(formWidth + 4).Align(lipgloss.Center).Render(btn))
	b.WriteString("\n\n")

	help := lipgloss.NewStyle().
		Width(w).
		Foreground(lipgloss.Color("#626262")).
		Background(lipgloss.Color("#1A1A1A")).
		Padding(0, 2).
		Render("↑↓/Tab navigate • Enter search • Ctrl+V paste • Esc back • Ctrl+C quit")
	b.WriteString(help)

	return b.String()
}

func (m SearchModel) submit() tea.Cmd {
	query := m.fields[0].value
	lang := m.fields[1].value
	if lang == "" {
		lang = "en"
	}

	return func() tea.Msg {
		return searchSubmitMsg{
			query:    query,
			lang:     lang,
			provider: m.providers[m.provider],
		}
	}
}
