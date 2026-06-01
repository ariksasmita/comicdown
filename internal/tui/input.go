package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// InputModel is Screen 1: URL + settings form.
type InputModel struct {
	focusIndex int
	inputs     []inputField
	opts       Opts
	err        string
	width      int
	height     int
}

type inputField struct {
	label       string
	value       string
	placeholder string
}

// NewInputModel creates the input form with defaults from opts.
func NewInputModel(opts Opts) InputModel {
	return InputModel{
		inputs: []inputField{
			{label: "Manga URL", value: opts.URL, placeholder: "https://mangadex.org/title/..."},
			{label: "Language", value: opts.Lang, placeholder: "en"},
			{label: "Quality (1-100)", value: fmt.Sprintf("%d", opts.Quality), placeholder: "85"},
			{label: "Max Width (px)", value: fmt.Sprintf("%d", opts.MaxWidth), placeholder: "1600"},
			{label: "Output Dir", value: opts.Output, placeholder: "./output"},
		},
		opts: opts,
	}
}

func (m InputModel) Init() tea.Cmd { return nil }

func (m InputModel) Update(msg tea.Msg) (InputModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "tab", "down":
			m.focusIndex = (m.focusIndex + 1) % len(m.inputs)
			return m, nil
		case "shift+tab", "up":
			m.focusIndex = (m.focusIndex - 1 + len(m.inputs)) % len(m.inputs)
			return m, nil
		case "enter":
			return m, m.submit()
		case "backspace":
			f := &m.inputs[m.focusIndex]
			if len(f.value) > 0 {
				f.value = f.value[:len(f.value)-1]
			}
			return m, nil
		default:
			if len(msg.String()) == 1 && msg.String()[0] >= 32 {
				m.inputs[m.focusIndex].value += msg.String()
			}
			return m, nil
		}

	case tea.PasteMsg:
		pasted := strings.ReplaceAll(msg.Content, "\n", "")
		pasted = strings.ReplaceAll(pasted, "\r", "")
		m.inputs[m.focusIndex].value += pasted
		return m, nil
	}

	return m, nil
}

func (m InputModel) View() string {
	w := m.width
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
		Render("📚  ComicDown — MangaDex CBZ Downloader")
	b.WriteString(title)
	b.WriteString("\n\n")

	// ── Form fields ──
	formWidth := min(w-8, 70)
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#B8B8B8")).
		Width(formWidth)
	focusedBorder := lipgloss.NewStyle().
		Width(formWidth).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7D56F4"))
	normalBorder := lipgloss.NewStyle().
		Width(formWidth).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3C3C3C"))

	for i, f := range m.inputs {
		border := normalBorder
		if i == m.focusIndex {
			border = focusedBorder
		}

		value := f.value
		if value == "" {
			value = lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).Render(f.placeholder)
		}

		b.WriteString("    ")
		b.WriteString(labelStyle.Render(f.label))
		b.WriteString("\n")
		rendered := border.Render("  " + value + "  ")
		lines := strings.Split(strings.TrimSuffix(rendered, "\n"), "\n")
		for _, line := range lines {
			b.WriteString("    " + line + "\n")
		}
	}

	// ── Submit button ──
	b.WriteString("\n")
	btn := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Bold(true).
		Padding(0, 3).
		Render("Fetch Chapters")
	b.WriteString(lipgloss.NewStyle().Width(formWidth + 4).Align(lipgloss.Center).Render(btn))
	b.WriteString("\n\n")

	// ── Help bar ──
	help := lipgloss.NewStyle().
		Width(w).
		Foreground(lipgloss.Color("#626262")).
		Background(lipgloss.Color("#1A1A1A")).
		Padding(0, 2).
		Render("↑↓/Tab  navigate  •  Enter  confirm  •  Ctrl+V  paste  •  Ctrl+C  quit")
	b.WriteString(help)

	if m.err != "" {
		b.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B6B")).Bold(true).Render("  ✗ "+m.err))
	}

	return b.String()
}

func (m InputModel) submit() tea.Cmd {
	opts := Opts{
		URL:     m.inputs[0].value,
		Lang:    m.inputs[1].value,
		Output:  m.inputs[4].value,
		Workers: 3,
	}
	fmt.Sscanf(m.inputs[2].value, "%d", &opts.Quality)
	fmt.Sscanf(m.inputs[3].value, "%d", &opts.MaxWidth)

	if opts.Quality <= 0 {
		opts.Quality = 85
	}
	if opts.Lang == "" {
		opts.Lang = "en"
	}
	if opts.Output == "" {
		opts.Output = "./output"
	}

	return func() tea.Msg { return inputDoneMsg{opts: opts} }
}
