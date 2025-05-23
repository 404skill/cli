package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

	downloadedStyle = lipgloss.NewStyle().
			Foreground(secondary).
			Faint(true).
			Render("✓ Downloaded")

	bubbleTableColumns = []btable.Column{
		btable.NewColumn("name", "Name", 32),
		btable.NewColumn("lang", "Language", 15),
		btable.NewColumn("diff", "Difficulty", 15),
		btable.NewColumn("dur", "Duration", 15),
		btable.NewColumn("status", "Status", 15),
	}
)

// --- State Machine ---
type tuiState int

const (
	stateRefreshingToken tuiState = iota
	stateMainMenu
	stateLogin
	stateProjectList
	stateLanguageSelection
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

	// Language Selection
	selectedProject *api.Project
	languages       []string
	languageIndex   int
	cloning         bool
	cloneProgress   float64
}

type errMsg struct {
	err error
}

// cloneCompleteMsg is sent when the git clone operation completes successfully
type cloneCompleteMsg struct{}

// cloneProgressMsg contains the current progress of the git clone operation
type cloneProgressMsg struct {
	progress float64
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
					// Check if project is already downloaded
					cfg, err := config.ReadConfig()
					if err == nil && cfg.DownloadedProjects != nil {
						if projectID, ok := selectedRow.Data["id"].(string); ok && cfg.DownloadedProjects[projectID] {
							m.errorMsg = "Project already downloaded. Please select a different project."
							return m, nil
						}
					}

					// Find the selected project
					for _, p := range m.projects {
						if p.ID == selectedRow.Data["id"] {
							m.selectedProject = &p
							// Split languages by comma and trim spaces
							m.languages = strings.Split(p.Language, ",")
							for i := range m.languages {
								m.languages[i] = strings.TrimSpace(m.languages[i])
							}
							m.languageIndex = 0
							m.state = stateLanguageSelection
							return m, nil
						}
					}
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

			// Get downloaded projects from config
			cfg, _ := config.ReadConfig()
			downloadedProjects := make(map[string]bool)
			if cfg.DownloadedProjects != nil {
				downloadedProjects = cfg.DownloadedProjects
			}

			for _, p := range msg {
				status := ""
				if downloadedProjects[p.ID] {
					status = "✓ Downloaded"
				}
				rows = append(rows, btable.NewRow(map[string]interface{}{
					"id":     p.ID,
					"name":   p.Name,
					"lang":   p.Language,
					"diff":   p.Difficulty,
					"dur":    fmt.Sprintf("%d min", p.EstimatedDurationInMinutes),
					"status": status,
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
	case stateLanguageSelection:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c", "q":
				m.quitting = true
				return m, tea.Quit
			case "esc", "b":
				m.state = stateProjectList
				m.selectedProject = nil
				m.languages = nil
				m.languageIndex = 0
				m.errorMsg = ""
				return m, nil
			case "up", "k":
				m.languageIndex--
				if m.languageIndex < 0 {
					m.languageIndex = len(m.languages) - 1
				}
			case "down", "j":
				m.languageIndex++
				if m.languageIndex >= len(m.languages) {
					m.languageIndex = 0
				}
			case "enter":
				if m.selectedProject != nil {
					m.cloning = true
					m.errorMsg = ""
					return m, tea.Batch(
						m.cloneProject(m.selectedProject.Name, m.languages[m.languageIndex]),
						m.updateCloneProgress(),
					)
				}
			}
		case cloneCompleteMsg:
			m.cloning = false
			m.state = stateProjectList
			m.selectedProject = nil
			m.languages = nil
			m.languageIndex = 0
			return m, nil
		case cloneProgressMsg:
			m.cloneProgress = msg.progress
			return m, nil
		case errMsg:
			m.errorMsg = msg.err.Error()
			m.cloning = false
			return m, nil
		}
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
		helpView := helpStyle.Render(m.help.View(keys) + "  [esc/b] back")
		info := ""
		if m.selectedInfo != "" {
			info = "\n" + m.selectedInfo
		}
		view := fmt.Sprintf("%s\n%s%s", m.table.View(), helpView, info)
		if m.errorMsg != "" {
			view = fmt.Sprintf("%s\n\n%s", view, errorStyle.Render(m.errorMsg))
		}
		return view
	case stateLanguageSelection:
		if m.cloning {
			progress := int(m.cloneProgress * 100)
			progressBar := strings.Repeat("█", progress/10) + strings.Repeat("░", 10-progress/10)
			return fmt.Sprintf("%s\n\nCloning project...\n[%s] %d%%\n\nPress q to quit",
				headerStyle.Render("Cloning Project"),
				progressBar,
				progress)
		}

		menu := headerStyle.Render("\nSelect a language for "+m.selectedProject.Name) + "\n\n"
		for i, lang := range m.languages {
			cursor := "  "
			style := menuItemStyle
			if m.languageIndex == i {
				cursor = "> "
				style = selectedMenuItemStyle
			}
			menu += fmt.Sprintf("%s%s\n", cursor, style.Render(lang))
		}
		menu += helpStyle.Render("\nUse ↑/↓ or k/j to move, Enter to select, [esc/b] back, q to quit")

		if m.errorMsg != "" {
			menu += "\n\n" + errorStyle.Render("Error: "+m.errorMsg)
		}
		return menu
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

// --- Project Cloning ---
// cloneProject initiates the git clone operation for the selected project and language.
// It creates the projects directory if it doesn't exist, formats the repository URL,
// and updates the config file with the downloaded project information.
func (m model) cloneProject(projectName, language string) tea.Cmd {
	return func() tea.Msg {
		// Create projects directory if it doesn't exist
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return errMsg{err: fmt.Errorf("failed to get home directory: %w", err)}
		}

		projectsDir := filepath.Join(homeDir, "404skill_projects")
		if err := os.MkdirAll(projectsDir, 0755); err != nil {
			return errMsg{err: fmt.Errorf("failed to create projects directory: %w", err)}
		}

		// Format project name for repo URL
		repoName := strings.ToLower(strings.ReplaceAll(projectName, " ", "_"))
		repoURL := fmt.Sprintf("https://github.com/404skill/%s_%s", repoName, language)
		targetDir := filepath.Join(projectsDir, fmt.Sprintf("%s_%s", repoName, language))

		// Start git clone
		cmd := exec.Command("git", "clone", repoURL, targetDir)
		if err := cmd.Run(); err != nil {
			return errMsg{err: fmt.Errorf("failed to clone repository: %w", err)}
		}

		// Update config with downloaded project
		cfg, err := config.ReadConfig()
		if err != nil {
			return errMsg{err: fmt.Errorf("failed to read config: %w", err)}
		}

		// Add project to downloaded projects list
		if cfg.DownloadedProjects == nil {
			cfg.DownloadedProjects = make(map[string]bool)
		}
		cfg.DownloadedProjects[m.selectedProject.ID] = true

		if err := config.WriteConfig(cfg); err != nil {
			return errMsg{err: fmt.Errorf("failed to update config: %w", err)}
		}

		return cloneCompleteMsg{}
	}
}

// updateCloneProgress simulates progress updates for the git clone operation.
// Since git clone doesn't provide real-time progress information, this function
// simulates progress to provide visual feedback to the user.
func (m model) updateCloneProgress() tea.Cmd {
	return func() tea.Msg {
		// Simulate progress updates
		for i := 0; i <= 100; i += 10 {
			time.Sleep(100 * time.Millisecond)
			return cloneProgressMsg{progress: float64(i) / 100}
		}
		return cloneCompleteMsg{}
	}
}
