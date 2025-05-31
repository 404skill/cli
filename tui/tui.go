package tui

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

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

// --- State Machine ---
type tuiState int

const (
	stateRefreshingToken tuiState = iota
	stateMainMenu
	stateLogin
	stateProjectList
	stateLanguageSelection
	stateConfirmRedownload
	stateTestProject
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

	// Components
	loginComponent    *LoginComponent
	projectComponent  *ProjectComponent
	languageComponent *LanguageComponent
	testComponent     *TestComponent

	// Dependencies
	fileManager   FileManager
	configManager ConfigManager
	help          help.Model

	// State
	ready    bool
	quitting bool
	errorMsg string

	// Login
	loginInputs   []textinput.Model
	loginFocusIdx int
	loginError    string
	loggingIn     bool

	// Projects
	table        btable.Model
	viewport     viewport.Model
	client       api.ClientInterface
	projects     []api.Project
	selected     int
	loading      bool
	selectedInfo string

	// Language Selection
	selectedProject *api.Project
	languages       []string
	languageIndex   int
	cloning         bool
	cloneProgress   float64

	// Confirm Redownload
	confirmRedownloadProject *api.Project
	confirmRedownloadLang    string
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

	fileManager := NewDefaultFileManager()
	configManager := config.NewConfigManager()

	state := stateLogin
	cfg, err := config.ReadConfig()
	if err == nil && cfg.Username != "" && cfg.Password != "" {
		state = stateRefreshingToken
	}

	return model{
		state:           state,
		mainMenuIndex:   0,
		mainMenuChoices: []string{"Download a project", "Test a project"},
		loginInputs:     []textinput.Model{username, password},
		loginFocusIdx:   0,
		table:           table,
		help:            help.New(),
		client:          client,
		selected:        -1,
		loading:         false,
		fileManager:     fileManager,
		configManager:   configManager,
		testComponent:   NewTestComponent(fileManager, configManager, client),
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
				if m.selectedAction == choiceTest {
					m.state = stateTestProject
					m.loading = true
					return m, fetchProjects(m.client)
				} else {
					m.state = stateProjectList
					m.loading = true
					return m, fetchProjects(m.client)
				}
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
							// Try to open the project directory
							homeDir, err := os.UserHomeDir()
							if err != nil {
								m.errorMsg = "Project already downloaded but couldn't determine home directory."
								return m, nil
							}

							// Find the project in our list to get its name
							var projectName string
							for _, p := range m.projects {
								if p.ID == projectID {
									projectName = p.Name
									break
								}
							}

							if projectName == "" {
								m.errorMsg = "Project already downloaded but couldn't find project details."
								return m, nil
							}

							// Format project name for directory
							repoName := strings.ToLower(strings.ReplaceAll(projectName, " ", "_"))
							projectsDir := filepath.Join(homeDir, "404skill_projects")

							// Try to find the project directory
							entries, err := os.ReadDir(projectsDir)
							if err != nil {
								m.errorMsg = "Project already downloaded but couldn't access projects directory."
								return m, nil
							}

							var projectDir string
							for _, entry := range entries {
								if entry.IsDir() && strings.HasPrefix(entry.Name(), repoName) {
									projectDir = filepath.Join(projectsDir, entry.Name())
									break
								}
							}

							if projectDir == "" {
								// Find the project and its languages
								for _, p := range m.projects {
									if p.ID == projectID {
										m.confirmRedownloadProject = &p
										m.languages = strings.Split(p.Language, ",")
										for i := range m.languages {
											m.languages[i] = strings.TrimSpace(m.languages[i])
										}
										m.languageIndex = 0
										m.state = stateConfirmRedownload
										return m, nil
									}
								}
								m.errorMsg = "Project was downloaded but directory not found. It might have been moved or deleted."
								return m, nil
							}

							// Try to open the directory
							if err := openFileExplorer(projectDir); err != nil {
								m.errorMsg = fmt.Sprintf("Project was downloaded but couldn't open directory: %v", err)
								return m, nil
							}

							m.errorMsg = "Project already downloaded. Opening project directory..."
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
					return m, m.cloneProject(m.selectedProject.Name, m.languages[m.languageIndex])
				}
			}
		case cloneCompleteMsg:
			m.cloning = false
			m.state = stateProjectList
			// Update the status of the cloned project in the table
			rows := []btable.Row{}
			for _, p := range m.projects {
				status := ""
				if (m.selectedProject != nil && p.ID == m.selectedProject.ID) ||
					(m.confirmRedownloadProject != nil && p.ID == m.confirmRedownloadProject.ID) {
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
			m.table = m.table.WithRows(rows)
			// Clear both project references
			m.selectedProject = nil
			m.confirmRedownloadProject = nil
			m.confirmRedownloadLang = ""
			return m, nil
		case cloneProgressMsg:
			m.cloneProgress = msg.progress
			return m, nil
		case errMsg:
			m.errorMsg = msg.err.Error()
			m.cloning = false
			return m, nil
		}
	case stateConfirmRedownload:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c", "q":
				m.quitting = true
				return m, tea.Quit
			case "esc", "b":
				m.state = stateProjectList
				m.confirmRedownloadProject = nil
				m.confirmRedownloadLang = ""
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
				if m.confirmRedownloadProject != nil {
					m.cloning = true
					m.errorMsg = ""
					return m, m.cloneProject(m.confirmRedownloadProject.Name, m.languages[m.languageIndex])
				}
			}
		case cloneCompleteMsg:
			m.cloning = false
			m.state = stateProjectList
			// Update the status of the cloned project in the table
			rows := []btable.Row{}
			for _, p := range m.projects {
				status := ""
				if (m.selectedProject != nil && p.ID == m.selectedProject.ID) ||
					(m.confirmRedownloadProject != nil && p.ID == m.confirmRedownloadProject.ID) {
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
			m.table = m.table.WithRows(rows)
			// Clear both project references
			m.selectedProject = nil
			m.confirmRedownloadProject = nil
			m.confirmRedownloadLang = ""
			return m, nil
		case cloneProgressMsg:
			m.cloneProgress = msg.progress
			return m, nil
		case errMsg:
			m.errorMsg = msg.err.Error()
			m.cloning = false
			return m, nil
		}
	case stateTestProject:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c", "q":
				m.quitting = true
				return m, tea.Quit
			case "esc", "b":
				m.state = stateMainMenu
				m.errorMsg = ""
				return m, nil
			}
		case []api.Project:
			// Pass projects to test component
			updatedComponent, cmd := m.testComponent.Update(msg)
			m.testComponent = updatedComponent.(*TestComponent)
			m.loading = false
			return m, cmd
		case errMsg:
			m.errorMsg = msg.err.Error()
			m.loading = false
			return m, nil
		}

		// Update test component
		updatedComponent, cmd := m.testComponent.Update(msg)
		m.testComponent = updatedComponent.(*TestComponent)
		return m, cmd
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
	case stateConfirmRedownload:
		if m.cloning {
			progress := int(m.cloneProgress * 100)
			progressBar := strings.Repeat("█", progress/10) + strings.Repeat("░", 10-progress/10)
			return fmt.Sprintf("%s\n\nCloning project...\n[%s] %d%%\n\nPress q to quit",
				headerStyle.Render("Cloning Project"),
				progressBar,
				progress)
		}

		menu := headerStyle.Render("\nProject directory not found. Would you like to re-download?") + "\n\n"
		menu += fmt.Sprintf("Project: %s\n\n", m.confirmRedownloadProject.Name)
		menu += "Select language:\n\n"
		for i, lang := range m.languages {
			cursor := "  "
			style := menuItemStyle
			if m.languageIndex == i {
				cursor = "> "
				style = selectedMenuItemStyle
			}
			menu += fmt.Sprintf("%s%s\n", cursor, style.Render(lang))
		}
		menu += helpStyle.Render("\nUse ↑/↓ or k/j to move, Enter to confirm, [esc/b] back, q to quit")

		if m.errorMsg != "" {
			menu += "\n\n" + errorStyle.Render("Error: "+m.errorMsg)
		}
		return menu
	case stateTestProject:
		if m.loading {
			return headerStyle.Render("\nLoading projects...")
		}
		return m.testComponent.View()
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

		// Read existing config to preserve DownloadedProjects
		cfg, err := config.ReadConfig()
		if err != nil {
			// If config doesn't exist, create new one
			cfg = config.Config{}
		}

		// Update only the auth-related fields
		cfg.Username = username
		cfg.Password = password
		cfg.AccessToken = token
		cfg.LastUpdated = time.Now()

		// Ensure DownloadedProjects map exists
		if cfg.DownloadedProjects == nil {
			cfg.DownloadedProjects = make(map[string]bool)
		}

		if err := config.WriteConfig(cfg); err != nil {
			return errMsg{err: fmt.Errorf("failed to write config: %w", err)}
		}
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

// --- Project Cloning ---
// openFileExplorer opens the file explorer at the specified path
func openFileExplorer(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start()
}

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

		testRepoUrl := fmt.Sprintf("https://github.com/404skill/%s_%s_test", repoName, language)
		testDir := filepath.Join(projectsDir, ".tests", fmt.Sprintf("%s_%s", repoName, language))
		if err := os.MkdirAll(testDir, 0755); err != nil {
			return errMsg{err: fmt.Errorf("failed to create tests directory: %w", err)}
		}

		// Start git clone with progress output
		cmdCloneProject := exec.Command("git", "clone", "--progress", repoURL, targetDir)
		cmdCloneTest := exec.Command("git", "clone", "--progress", testRepoUrl, testDir)

		stderr, err := cmdCloneProject.StderrPipe()
		if err != nil {
			return errMsg{err: fmt.Errorf("failed to create stderr pipe: %w", err)}
		}

		if err := cmdCloneProject.Start(); err != nil {
			return errMsg{err: fmt.Errorf("failed to start git clone: %w", err)}
		}

		if err := cmdCloneTest.Start(); err != nil {
			return errMsg{err: fmt.Errorf("failed to start git clone: %w", err)}
		}

		// Read progress from stderr
		scanner := bufio.NewScanner(stderr)
		var cloneError string
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "Receiving objects") {
				// Parse percentage from line like "Receiving objects: 45% (9/20)"
				if strings.Contains(line, "%") {
					parts := strings.Split(line, "%")
					if len(parts) > 0 {
						if progress, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64); err == nil {
							// Send progress update
							tea.Batch(
								func() tea.Msg { return cloneProgressMsg{progress: progress / 100} },
							)()
						}
					}
				}
			} else if strings.Contains(line, "error:") || strings.Contains(line, "fatal:") {
				cloneError = line
			}
		}

		if err := cmdCloneProject.Wait(); err != nil {
			if cloneError != "" {
				return errMsg{err: fmt.Errorf("git clone failed: %s", cloneError)}
			}
			return errMsg{err: fmt.Errorf("git clone failed: %w", err)}
		}

		if err := cmdCloneTest.Wait(); err != nil {
			return errMsg{err: fmt.Errorf("git clone failed: %w", err)}
		}

		// Verify the clone was successful by checking if the directory exists and has content
		if _, err := os.Stat(targetDir); os.IsNotExist(err) {
			return errMsg{err: fmt.Errorf("clone appeared to succeed but target directory is missing")}
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

		// Get the project ID based on the current state
		var projectID string
		if m.state == stateConfirmRedownload && m.confirmRedownloadProject != nil {
			projectID = m.confirmRedownloadProject.ID
		} else if m.selectedProject != nil {
			projectID = m.selectedProject.ID
		} else {
			return errMsg{err: fmt.Errorf("no project selected for download")}
		}

		cfg.DownloadedProjects[projectID] = true

		if err := config.WriteConfig(cfg); err != nil {
			return errMsg{err: fmt.Errorf("failed to update config: %w", err)}
		}

		// Open file explorer at the cloned directory
		if err := openFileExplorer(targetDir); err != nil {
			// Don't return error here, as the clone was successful
			// Just log the error and continue
			fmt.Printf("Warning: Failed to open file explorer: %v\n", err)
		}

		if err := m.client.InitializeProject(context.Background(), projectID); err != nil {
			return errMsg{err: fmt.Errorf("failed to update profile project. error: %w", err)}
		}

		return cloneCompleteMsg{}
	}
}
