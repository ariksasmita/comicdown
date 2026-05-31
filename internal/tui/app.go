package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// screen identifies which screen is currently active.
type screen int

const (
	screenInput    screen = iota // URL + settings form
	screenLoading                // Fetching chapters from API
	screenChapters               // Chapter selector
	screenProgress               // Download/optimize/package progress
	screenSummary                // Results summary
	screenError                  // Fatal error display
)

// Opts holds the user's download configuration.
type Opts struct {
	URL       string
	Lang      string
	Quality   int
	MaxWidth  int
	Output    string
	Workers   int
	DataSaver bool
}

// Model is the root Bubble Tea model. It delegates to the active screen.
type Model struct {
	current screen
	width   int
	height  int

	// Sub-models for each screen.
	input    InputModel
	chapters ChaptersModel
	progress ProgressModel
	summary  SummaryModel

	// Shared state.
	opts   Opts
	errMsg string
}

// NewModel returns the initial TUI model (starts on input screen).
func NewModel() Model {
	return Model{
		current: screenInput,
		opts: Opts{
			Lang:     "en",
			Quality:  85,
			MaxWidth: 1600,
			Output:   "./output",
			Workers:  3,
		},
		input: NewInputModel(Opts{
			Lang:     "en",
			Quality:  85,
			MaxWidth: 1600,
			Output:   "./output",
			Workers:  3,
		}),
	}
}

func (m Model) Init() tea.Cmd {
	return m.input.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	switch m.current {
	case screenInput:
		return m.updateInput(msg)
	case screenLoading:
		return m.updateLoading(msg)
	case screenChapters:
		return m.updateChapters(msg)
	case screenProgress:
		return m.updateProgress(msg)
	case screenSummary:
		return m.updateSummary(msg)
	case screenError:
		return m.updateError(msg)
	default:
		return m, nil
	}
}

func (m Model) View() tea.View {
	var s string
	switch m.current {
	case screenInput:
		s = m.input.View()
	case screenLoading:
		s = m.viewLoading()
	case screenChapters:
		s = m.chapters.View(m.width)
	case screenProgress:
		s = m.progress.View(m.width)
	case screenSummary:
		s = m.summary.View(m.width)
	case screenError:
		s = m.viewError()
	}
	return tea.NewView(s)
}

// --- Internal messages for screen transitions ---

type inputDoneMsg struct{ opts Opts }
type chaptersLoadedMsg struct {
	manga    mangaInfo
	chapters []chapterInfo
}
type chaptersLoadErrMsg struct{ err error }
type downloadStartMsg struct{}
type downloadDoneMsg struct{ summary SummaryModel }
type backToInputMsg struct{}

// Lightweight structs to pass data between screens without import cycles.
type mangaInfo struct {
	Title string
	Tags  []string
	Year  int
}

type chapterInfo struct {
	ID                 string
	Volume             string
	Chapter            string
	Title              string
	TranslatedLanguage string
	Pages              int
}

// --- Screen update methods ---

func (m Model) updateInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)

	if _, ok := msg.(inputDoneMsg); ok {
		m.opts = msg.(inputDoneMsg).opts
		m.current = screenLoading
		return m, fetchChaptersCmd(m.opts)
	}

	return m, cmd
}

func (m Model) updateLoading(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case chaptersLoadedMsg:
		m.chapters = NewChaptersModel(msg.manga, msg.chapters, m.width, m.height)
		m.current = screenChapters
		return m, m.chapters.Init()
	case chaptersLoadErrMsg:
		m.errMsg = msg.err.Error()
		m.current = screenError
		return m, nil
	}
	return m, nil
}

func (m Model) updateChapters(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case downloadStartMsg:
		selected := m.chapters.Selected()
		m.progress = NewProgressModel(selected, m.opts, m.width, m.height, m.chapters.manga)
		m.current = screenProgress
		return m, m.progress.Init()
	case backToInputMsg:
		m.current = screenInput
		return m, nil
	}

	var cmd tea.Cmd
	m.chapters, cmd = m.chapters.Update(msg)
	return m, cmd
}

func (m Model) updateProgress(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case downloadDoneMsg:
		m.summary = msg.summary
		m.current = screenSummary
		return m, nil
	}

	var cmd tea.Cmd
	m.progress, cmd = m.progress.Update(msg)
	return m, cmd
}

func (m Model) updateSummary(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case backToInputMsg:
		m.current = screenInput
		m.input = NewInputModel(m.opts)
		return m, m.input.Init()
	}

	var cmd tea.Cmd
	m.summary, cmd = m.summary.Update(msg)
	return m, cmd
}

func (m Model) updateError(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "enter" || msg.String() == "esc" {
			m.current = screenInput
			return m, nil
		}
	}
	return m, nil
}

// --- Shared styled views ---

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 2)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6B6B")).
			Bold(true)

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))
)

func (m Model) viewLoading() string {
	return fmt.Sprintf(
		"\n  %s\n\n  %s\n",
		titleStyle.Render("📚 ComicDown"),
		"Fetching chapters from MangaDex...",
	)
}

func (m Model) viewError() string {
	return fmt.Sprintf(
		"\n  %s\n\n  %s\n\n  %s\n",
		titleStyle.Render("❌ Error"),
		errorStyle.Render(m.errMsg),
		dimStyle.Render("Press Enter or Esc to go back"),
	)
}
