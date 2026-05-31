package tui

import (
	"context"
	"fmt"

	"github.com/sasmitai/comicdown/internal/mangadex"
	"github.com/sasmitai/comicdown/internal/provider"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// screen identifies which screen is currently active.
type screen int

const (
	screenHome     screen = iota // Choose: paste URL or search
	screenInput                   // Paste URL + settings
	screenSearch                  // Search by title form
	screenResults                 // Search results list
	screenLoading                 // Fetching chapters from API
	screenChapters                // Chapter selector
	screenProgress                // Download/optimize/package progress
	screenSummary                 // Results summary
	screenError                   // Fatal error display
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
	home     HomeModel
	input    InputModel
	search   SearchModel
	results  ResultsModel
	chapters ChaptersModel
	progress ProgressModel
	summary  SummaryModel

	// Shared state.
	prov    provider.Provider
	opts    Opts
	errMsg  string
}

// NewModel returns the initial TUI model (starts on home screen).
func NewModel() Model {
	opts := Opts{
		Lang:     "en",
		Quality:  85,
		MaxWidth: 1600,
		Output:   "./output",
		Workers:  3,
	}
	return Model{
		current: screenHome,
		opts:    opts,
		home:    NewHomeModel(),
		input:   NewInputModel(opts),
		search:  NewSearchModel(),
	}
}

func (m Model) Init() tea.Cmd {
	return m.home.Init()
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
	case screenHome:
		return m.updateHome(msg)
	case screenInput:
		return m.updateInput(msg)
	case screenSearch:
		return m.updateSearch(msg)
	case screenResults:
		return m.updateResults(msg)
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
	case screenHome:
		s = m.home.View()
	case screenInput:
		s = m.input.View()
	case screenSearch:
		s = m.search.View()
	case screenResults:
		s = m.results.View(m.width)
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

type homeSelectMsg struct{ mode entryMode }
type backToHomeMsg struct{}
type backToSearchMsg struct{}
type searchSubmitMsg struct {
	query    string
	lang     string
	provider string
}
type searchResultsMsg struct {
	results  []provider.SearchResult
	query    string
	provider string
}
type searchErrMsg struct{ err error }
type resultSelectMsg struct{ result provider.SearchResult }
type inputDoneMsg struct{ opts Opts }
type chaptersLoadedMsg struct {
	manga    mangaInfo
	chapters []chapterInfo
}
type chaptersLoadErrMsg struct{ err error }
type downloadStartMsg struct{}
type downloadDoneMsg struct{ summary SummaryModel }
type backToInputMsg struct{}

// Lightweight structs to pass data between screens.
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

func (m Model) updateHome(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.home, cmd = m.home.Update(msg)

	if h, ok := msg.(homeSelectMsg); ok {
		switch h.mode {
		case entryPasteURL:
			m.current = screenInput
			return m, m.input.Init()
		case entrySearch:
			m.current = screenSearch
			return m, m.search.Init()
		}
	}

	return m, cmd
}

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

func (m Model) updateSearch(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)

	switch msg := msg.(type) {
	case searchSubmitMsg:
		m.current = screenLoading
		return m, searchMangaCmd(msg.query, msg.lang)
	case backToHomeMsg:
		m.current = screenHome
		return m, nil
	}

	return m, cmd
}

func (m Model) updateResults(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case resultSelectMsg:
		// Fetch full manga + chapters for selected result
		m.current = screenLoading
		return m, fetchMangaFromResultCmd(msg.result, m.search.fields[1].value)
	case backToSearchMsg:
		m.current = screenSearch
		return m, nil
	}

	var cmd tea.Cmd
	m.results, cmd = m.results.Update(msg)
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
	case searchResultsMsg:
		m.results = NewResultsModel(msg.results, msg.query, msg.provider, m.width, m.height)
		m.current = screenResults
		return m, nil
	case searchErrMsg:
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
	switch msg.(type) {
	case downloadDoneMsg:
		m.summary = msg.(downloadDoneMsg).summary
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
		m.current = screenHome
		m.home = NewHomeModel()
		m.input = NewInputModel(m.opts)
		return m, m.home.Init()
	}

	var cmd tea.Cmd
	m.summary, cmd = m.summary.Update(msg)
	return m, cmd
}

func (m Model) updateError(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "enter" || msg.String() == "esc" {
			m.current = screenHome
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
		"Fetching data...",
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

// --- Async commands ---

// fetchChaptersCmd fetches manga + chapters from a pasted MangaDex URL.
func fetchChaptersCmd(opts Opts) tea.Cmd {
	return func() tea.Msg {
		c := mangadex.NewClient()

		id, err := c.ExtractID(opts.URL)
		if err != nil {
			return chaptersLoadErrMsg{err: fmt.Errorf("invalid URL: %w", err)}
		}

		manga, err := c.FetchManga(context.Background(), id)
		if err != nil {
			return chaptersLoadErrMsg{err: fmt.Errorf("fetch manga: %w", err)}
		}

		chapters, err := c.FetchAllChapters(context.Background(), id, opts.Lang)
		if err != nil {
			return chaptersLoadErrMsg{err: fmt.Errorf("fetch chapters: %w", err)}
		}

		if len(chapters) == 0 {
			return chaptersLoadErrMsg{err: fmt.Errorf("no %s chapters found for this manga", opts.Lang)}
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

// searchMangaCmd searches for manga by title.
func searchMangaCmd(query, lang string) tea.Cmd {
	return func() tea.Msg {
		c := mangadex.NewClient()
		results, err := c.Search(context.Background(), query, provider.SearchOpts{
			Language: lang,
			Limit:    25,
		})
		if err != nil {
			return searchErrMsg{err: err}
		}
		return searchResultsMsg{
			results:  results,
			query:    query,
			provider: "MangaDex",
		}
	}
}

// fetchMangaFromResultCmd fetches full manga + chapters for a search result selection.
func fetchMangaFromResultCmd(r provider.SearchResult, lang string) tea.Cmd {
	return func() tea.Msg {
		c := mangadex.NewClient()

		manga, err := c.FetchManga(context.Background(), r.ID)
		if err != nil {
			return chaptersLoadErrMsg{err: fmt.Errorf("fetch manga: %w", err)}
		}

		chapters, err := c.FetchAllChapters(context.Background(), r.ID, lang)
		if err != nil {
			return chaptersLoadErrMsg{err: fmt.Errorf("fetch chapters: %w", err)}
		}

		if len(chapters) == 0 {
			return chaptersLoadErrMsg{err: fmt.Errorf("no %s chapters found for this manga", lang)}
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
