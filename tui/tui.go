package tui

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"404skill-cli/api"
	"404skill-cli/auth"
	"404skill-cli/config"
	"404skill-cli/supabase"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	btable "github.com/evertras/bubble-table/table"
)

// --- Styling ---
var (
	// Colors
	primary    = lipgloss.Color("#00ff00") // Bright green
	secondary  = lipgloss.Color("#00aa00") // Darker green
	accent     = lipgloss.Color("#00ffaa") // Cyan-green
	errorColor = lipgloss.Color("#ff0000") // Red
	bg         = lipgloss.Color("#000000") // Black

	// Styles
	baseStyle = lipgloss.NewStyle().
			Foreground(primary).
			Background(bg).
			Padding(1, 2)

	headerStyle = lipgloss.NewStyle().
			Foreground(accent).
			Bold(true).
			Underline(true).
			Padding(0, 1)

	menuStyle = lipgloss.NewStyle().
			Foreground(primary).
			Background(bg).
			Padding(0, 1)

	menuItemStyle = lipgloss.NewStyle().
			Foreground(primary).
			Background(bg).
			Padding(0, 1)

	selectedMenuItemStyle = lipgloss.NewStyle().
				Foreground(bg).
				Background(primary).
				Bold(true).
				Padding(0, 1)

	loginBoxStyle = lipgloss.NewStyle().
			Foreground(primary).
			Background(bg).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(1, 4).
			Width(44)

	errorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(secondary).
			Faint(true)

	asciiArt = lipgloss.NewStyle().
			Foreground(primary).Render(`
/==============================================================================================\
||                                                                                            ||
||      ___   ___  ________  ___   ___  ________  ___  __    ___  ___       ___               ||
||     |\  \ |\  \|\   __  \|\  \ |\  \|\   ____\|\  \|\  \ |\  \|\  \     |\  \              ||
||     \ \  \\_\  \ \  \|\  \ \  \\_\  \ \  \___|\ \  \/  /|\ \  \ \  \    \ \  \             ||
||      \ \______  \ \  \\\  \ \______  \ \_____  \ \   ___  \ \  \ \  \    \ \  \            ||
||       \|_____|\  \ \  \\\  \|_____|\  \|____|\  \ \  \\ \  \ \  \ \  \____\ \  \____       ||
||              \ \__\ \_______\     \ \__\____\_\  \ \__\\ \__\ \__\ \_______\ \_______\     ||
||               \|__|\|_______|      \|__|\_________\|__| \|__|\|__|\|_______|\|_______|     ||
||                                        \|_________|                                        ||
||                                                                                            ||
\==============================================================================================/
                                                                       `)

	bubbleTableColumns = []btable.Column{
		btable.NewColumn("id", "ID", 40),
		btable.NewColumn("name", "Name", 32),
		btable.NewColumn("lang", "Language", 15),
		btable.NewColumn("diff", "Difficulty", 15),
		btable.NewColumn("dur", "Duration", 15),
	}
)

// --- State Machine ---
type tuiState int

const (
	stateRefreshingToken tuiState = iota
	stateMainMenu
	stateLogin
	stateProjectList
)

type mainMenuChoice int

const (
	choiceInit mainMenuChoice = iota
	choiceTest
)

// --- Model ---
type model struct {
	// State
	state           tuiState
	mainMenuIndex   int
	mainMenuChoices []string
	selectedAction  mainMenuChoice

	// Login
	loginInputs   []textinput.Model
	loginFocusIdx int
	loginError    string
	loggingIn     bool

	// Projects
	table        btable.Model
	viewport     viewport.Model
	help         help.Model
	client       api.ClientInterface
	projects     []api.Project
	selected     int
	ready        bool
	quitting     bool
	errorMsg     string
	loading      bool
	selectedInfo string
}

type errMsg struct {
	err error
}

type keyMap struct {
	Up    key.Binding
	Down  key.Binding
	Enter key.Binding
	Quit  key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter},
		{k.Quit},
	}
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select/submit"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

// --- Initial Model ---
func InitialModel(client api.ClientInterface) model {
	// Login inputs
	username := textinput.New()
	username.Placeholder = "Username"
	username.Focus()
	username.CharLimit = 64
	username.Width = 32

	password := textinput.New()
	password.Placeholder = "Password"
	password.EchoMode = textinput.EchoPassword
	password.EchoCharacter = '•'
	password.CharLimit = 64
	password.Width = 32

	rows := []btable.Row{}
	table := btable.New(bubbleTableColumns).WithRows(rows)

	return model{
		state:           stateRefreshingToken,
		mainMenuIndex:   0,
		mainMenuChoices: []string{"Init a project", "Test a project"},
		loginInputs:     []textinput.Model{username, password},
		loginFocusIdx:   0,
		table:           table,
		help:            help.New(),
		client:          client,
		selected:        -1,
		loading:         false,
	}
}

// --- State Machine ---
func (m model) Init() tea.Cmd {
	cfg, err := config.ReadConfig()
	if err == nil && cfg.Username != "" && cfg.Password != "" {
		return refreshTokenCmd()
	}
	m.state = stateLogin
	return nil
}

// --- Token Refresh Command ---
type tokenRefreshMsg struct {
	err error
}

func refreshTokenCmd() tea.Cmd {
	return func() tea.Msg {
		_, err := config.NewConfigTokenProvider().GetToken()
		return tokenRefreshMsg{err: err}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.state {
	case stateRefreshingToken:
		switch msg := msg.(type) {
		case tokenRefreshMsg:
			if msg.err == nil {
				m.state = stateMainMenu
				return m, nil
			} else {
				m.state = stateLogin
				m.loginError = "Session expired. Please log in again."
				return m, nil
			}
		}
		// Block all other input
		return m, nil
	case stateMainMenu:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "up", "k":
				m.mainMenuIndex--
				if m.mainMenuIndex < 0 {
					m.mainMenuIndex = len(m.mainMenuChoices) - 1
				}
			case "down", "j":
				m.mainMenuIndex++
				if m.mainMenuIndex >= len(m.mainMenuChoices) {
					m.mainMenuIndex = 0
				}
			case "enter":
				m.selectedAction = mainMenuChoice(m.mainMenuIndex)
				m.state = stateProjectList
				m.loading = true
				return m, fetchProjects(m.client)
			case "q", "ctrl+c":
				m.quitting = true
				return m, tea.Quit
			}
		case string:
			if msg == "login-success" {
				m.state = stateMainMenu
				m.loginError = ""
				m.loggingIn = false
			}
		case errMsg:
			m.state = stateLogin
			m.loginError = msg.err.Error()
			m.loggingIn = false
		}
	case stateLogin:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c", "q":
				m.quitting = true
				return m, tea.Quit
			case "tab", "shift+tab":
				if msg.String() == "shift+tab" {
					m.loginFocusIdx--
				} else {
					m.loginFocusIdx++
				}
				if m.loginFocusIdx > 1 {
					m.loginFocusIdx = 0
				} else if m.loginFocusIdx < 0 {
					m.loginFocusIdx = 1
				}
				for i := 0; i < len(m.loginInputs); i++ {
					if i == m.loginFocusIdx {
						m.loginInputs[i].Focus()
					} else {
						m.loginInputs[i].Blur()
					}
				}
				return m, nil
			case "enter":
				if m.loginFocusIdx == 1 {
					m.loggingIn = true
					m.loginError = ""
					return m, m.tryLogin()
				}
				m.loginFocusIdx = 1
				for i := 0; i < len(m.loginInputs); i++ {
					if i == m.loginFocusIdx {
						m.loginInputs[i].Focus()
					} else {
						m.loginInputs[i].Blur()
					}
				}
				return m, nil
			default:
				// Pass all other keys to the focused input only
				m.loginInputs[m.loginFocusIdx], cmd = m.loginInputs[m.loginFocusIdx].Update(msg)
				return m, cmd
			}
		case string:
			if msg == "login-success" {
				m.state = stateMainMenu
				m.loginError = ""
				m.loggingIn = false
			}
		case errMsg:
			m.loginError = msg.err.Error()
			m.loggingIn = false
		}
	case stateProjectList:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c", "q":
				m.quitting = true
				return m, tea.Quit
			case "esc", "b":
				m.state = stateMainMenu
				m.errorMsg = ""
				m.selected = -1
				return m, nil
			case "enter":
				selectedRow := m.table.HighlightedRow()
				if selectedRow.Data != nil {
					id := selectedRow.Data["id"]
					name := selectedRow.Data["name"]
					lang := selectedRow.Data["lang"]
					diff := selectedRow.Data["diff"]
					dur := selectedRow.Data["dur"]
					m.selectedInfo = fmt.Sprintf("Selected: ID=%v, Name=%v, Language=%v, Difficulty=%v, Duration=%v", id, name, lang, diff, dur)
				}
			}
		case tea.WindowSizeMsg:
			if !m.ready {
				m.ready = true
				m.viewport = viewport.New(msg.Width, msg.Height-7)
				m.viewport.Style = baseStyle
			}
		case []api.Project:
			m.projects = msg
			rows := []btable.Row{}
			for _, p := range msg {
				rows = append(rows, btable.NewRow(map[string]interface{}{
					"id":   p.ID,
					"name": p.Name,
					"lang": p.Language,
					"diff": p.Difficulty,
					"dur":  fmt.Sprintf("%d min", p.EstimatedDurationInMinutes),
				}))
			}
			m.table = btable.New(bubbleTableColumns).
				WithRows(rows).
				Focused(true)

			m.loading = false
		case errMsg:
			m.errorMsg = msg.err.Error()
			m.loading = false
		}
		m.table, cmd = m.table.Update(msg)
	}

	return m, cmd
}

// --- View ---
func (m model) View() string {
	if m.quitting {
		return errorStyle.Render("Goodbye!") + "\n"
	}

	switch m.state {
	case stateRefreshingToken:
		return headerStyle.Render("\nRefreshing session... Please wait.")
	case stateMainMenu:
		menu := asciiArt + "\n"
		for i, choice := range m.mainMenuChoices {
			cursor := "  "
			style := menuItemStyle
			if m.mainMenuIndex == i {
				cursor = "> "
				style = selectedMenuItemStyle
			}
			menu += fmt.Sprintf("%s%s\n", cursor, style.Render(choice))
		}
		menu += helpStyle.Render("\nUse ↑/↓ or k/j to move, Enter to select, q to quit.")
		return menu
	case stateLogin:
		inputs := []string{}
		for i := range m.loginInputs {
			input := m.loginInputs[i].View()
			// Add a green blinking cursor to the focused input
			if i == m.loginFocusIdx {
				input += lipgloss.NewStyle().Foreground(accent).Render("█")
			}
			inputs = append(inputs, input)
		}
		loginBox := loginBoxStyle.Render(
			"Username: " + inputs[0] + "\n" +
				"Password: " + inputs[1] + "\n" +
				strings.Repeat(" ", 2) + "[Tab] Switch  [Enter] Submit  [q] Quit" +
				func() string {
					if m.loginError != "" {
						return "\n" + errorStyle.Render(m.loginError)
					}
					if m.loggingIn {
						return "\n" + headerStyle.Render("Logging in...")
					}
					return ""
				}(),
		)
		// Center the login box
		termWidth, termHeight := 80, 24
		if m.ready && m.viewport.Width > 0 && m.viewport.Height > 0 {
			termWidth, termHeight = m.viewport.Width, m.viewport.Height
		}
		boxLines := strings.Split(loginBox, "\n")
		boxHeight := len(boxLines)
		padTop := (termHeight - boxHeight) / 2
		padLeft := (termWidth - loginBoxStyle.GetWidth()) / 2
		centered := strings.Repeat("\n", padTop) +
			asciiArt + "\n\n" +
			strings.Repeat(" ", padLeft) + strings.Join(boxLines, "\n"+strings.Repeat(" ", padLeft))
		return centered
	case stateProjectList:
		if m.loading {
			return headerStyle.Render("\nLoading projects...")
		}
		if m.errorMsg != "" {
			return errorStyle.Render(fmt.Sprintf("Error: %s\nPress q to quit.", m.errorMsg))
		}
		helpView := helpStyle.Render(m.help.View(keys) + "  [esc/b] back")
		info := ""
		if m.selectedInfo != "" {
			info = "\n" + m.selectedInfo
		}
		return fmt.Sprintf("%s\n%s%s", m.table.View(), helpView, info)
	}
	return ""
}

// --- Login Logic ---
func (m model) tryLogin() tea.Cmd {
	return func() tea.Msg {
		username := m.loginInputs[0].Value()
		password := m.loginInputs[1].Value()
		client, err := supabase.NewSupabaseClient()
		if err != nil {
			return errMsg{err: fmt.Errorf("failed to create supabase client: %w", err)}
		}
		authProvider := auth.NewSupabaseAuth(client)
		token, err := authProvider.SignIn(context.Background(), username, password)
		if err != nil {
			return errMsg{err: fmt.Errorf("invalid credentials: %w", err)}
		}
		cfg := config.Config{
			Username:    username,
			Password:    password,
			AccessToken: token,
			LastUpdated: time.Now(),
		}
		_ = config.WriteConfig(cfg)
		return "login-success"
	}
}

// --- Fetch Projects ---
func fetchProjects(client api.ClientInterface) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		projects, err := client.ListProjects(ctx)
		if err != nil {
			return errMsg{err: err}
		}
		return projects
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func truncateString(s string, maxLen int) string {
	if utf8.RuneCountInString(s) <= maxLen {
		return s
	}
	runes := []rune(s)
	if maxLen > 3 {
		return string(runes[:maxLen-3]) + "..."
	}
	return string(runes[:maxLen])
}

func fitTableColumns(projects []api.Project) []btable.Column {
	headers := []string{"ID", "Name", "Language", "Difficulty", "Duration"}
	maxLens := make([]int, len(headers))
	for i, h := range headers {
		maxLens[i] = utf8.RuneCountInString(h)
	}
	for _, p := range projects {
		row := []string{
			p.ID,
			p.Name,
			p.Language,
			p.Difficulty,
			fmt.Sprintf("%d min", p.EstimatedDurationInMinutes),
		}
		for i, cell := range row {
			maxLens[i] = max(maxLens[i], utf8.RuneCountInString(cell))
		}
	}
	cols := make([]btable.Column, len(headers))
	for i, h := range headers {
		cols[i] = btable.NewColumn(h, h, maxLens[i]+2) // +2 for padding
	}
	return cols
}
