package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"404skill-cli/api"
	"404skill-cli/auth"
	"404skill-cli/config"
	"404skill-cli/downloader"
	"404skill-cli/filesystem"
	"404skill-cli/supabase"
	"404skill-cli/testrunner"
	"404skill-cli/tui/components/footer"
	"404skill-cli/tui/components/menu"
	"404skill-cli/tui/language"
	"404skill-cli/tui/login"
	"404skill-cli/tui/projects"
	"404skill-cli/tui/test"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
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
	loginComponent    *login.Component
	projectComponent  *projects.Component
	languageComponent *language.Component
	testComponent     test.Component

	// Menu components
	mainMenu *menu.Component

	// Dependencies
	fileManager   *filesystem.Manager
	configManager *config.ConfigManager
	help          help.Model
	footer        *footer.Component

	// State
	ready    bool
	quitting bool
	errorMsg string

	// Projects
	table        btable.Model
	viewport     viewport.Model
	client       api.ClientInterface
	projects     []api.Project
	selected     int
	loading      bool
	selectedInfo string

	// Downloader
	downloader downloader.Downloader
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
	rows := []btable.Row{}
	btableModel := btable.New(bubbleTableColumns).WithRows(rows)

	fileManager := filesystem.NewManager()
	configManager := config.NewConfigManager()

	// Create auth provider for dependency injection
	supabaseClient, err := supabase.NewSupabaseClient()
	if err != nil {
		// Handle error appropriately - for now we'll continue with nil
		// In production, you might want to handle this differently
	}
	authProvider := auth.NewSupabaseAuth(supabaseClient)

	state := stateLogin
	if configManager.HasCredentials() {
		state = stateRefreshingToken
	}

	// Create main menu with default theme styles
	mainMenu := menu.New([]string{"Download a project", "Test a project"})

	// Create downloader
	gitDownloader := downloader.NewGitDownloader(fileManager, configManager, client)

	m := model{
		state:            state,
		mainMenuIndex:    0,
		mainMenuChoices:  []string{"Download a project", "Test a project"},
		loginComponent:   login.New(authProvider, configManager),
		projectComponent: projects.New(client, configManager, fileManager),
		table:            btableModel,
		help:             help.New(),
		client:           client,
		selected:         -1,
		loading:          false,
		fileManager:      fileManager,
		configManager:    configManager,
		testComponent:    test.New(testrunner.NewDefaultTestRunner(), configManager, client),
		footer:           footer.New(),
		mainMenu:         mainMenu,
		downloader:       gitDownloader,
	}

	return m
}

// GetProjectStatus implements table.ProjectStatusProvider
func (m *model) GetProjectStatus(projectID string) string {
	if m.configManager.IsProjectDownloaded(projectID) {
		return "✓ Downloaded"
	}
	return ""
}

// --- State Machine ---
func (m model) Init() tea.Cmd {
	if m.configManager.HasCredentials() {
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
		configManager := config.NewConfigManager()
		_, err := configManager.GetToken()
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
				m.loginComponent.SetError("Session expired. Please log in again.")
				return m, nil
			}
		}
		// Block all other input
		return m, nil
	case stateMainMenu:
		// Update main menu component
		var menuCmd tea.Cmd
		m.mainMenu, menuCmd = m.mainMenu.Update(msg)

		switch msg := msg.(type) {
		case menu.MenuSelectMsg:
			m.selectedAction = mainMenuChoice(msg.SelectedIndex)
			if m.selectedAction == choiceTest {
				m.state = stateTestProject
				m.loading = true
				return m, fetchProjects(m.client)
			} else {
				m.state = stateProjectList
				m.loading = true
				return m, fetchProjects(m.client)
			}
		case tea.KeyMsg:
			switch msg.String() {
			case "q", "ctrl+c":
				m.quitting = true
				return m, tea.Quit
			}
		case login.LoginSuccessMsg:
			m.state = stateMainMenu
			return m, nil
		case login.LoginErrorMsg:
			m.state = stateLogin
			m.loginComponent.SetError(msg.Error)
			return m, nil
		case errMsg:
			m.state = stateLogin
			m.loginComponent.SetError(msg.err.Error())
			return m, nil
		}

		if menuCmd != nil {
			return m, menuCmd
		}
	case stateLogin:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c", "q":
				m.quitting = true
				return m, tea.Quit
			default:
				// Delegate all login input handling to the login component
				updatedComponent, cmd := m.loginComponent.Update(msg)
				m.loginComponent = updatedComponent
				return m, cmd
			}
		case login.LoginSuccessMsg:
			m.state = stateMainMenu
			return m, nil
		case login.LoginErrorMsg:
			m.loginComponent.SetError(msg.Error)
			return m, nil
		case errMsg:
			m.loginComponent.SetError(msg.err.Error())
			return m, nil
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
				selectedProject := m.projectComponent.GetSelectedProject()
				if selectedProject != nil {
					// Check if project is already downloaded
					if m.configManager.IsProjectDownloaded(selectedProject.ID) {
						// Try to open the project directory
						homeDir, err := os.UserHomeDir()
						if err != nil {
							m.errorMsg = "Project already downloaded but couldn't determine home directory."
							return m, nil
						}

						// Use the selected project's name
						projectName := selectedProject.Name

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
							// Project directory not found, go to language selection for re-download
							m.languageComponent = language.New(selectedProject, m.downloader)
							m.state = stateLanguageSelection
							return m, nil
						}

						// Try to open the directory
						if err := m.fileManager.OpenFileExplorer(projectDir); err != nil {
							m.errorMsg = fmt.Sprintf("Project was downloaded but couldn't open directory: %v", err)
							return m, nil
						}

						m.errorMsg = "Project already downloaded. Opening project directory..."
						return m, nil
					}

					// Create language component for new download
					m.languageComponent = language.New(selectedProject, m.downloader)
					m.state = stateLanguageSelection
					return m, nil
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
			m.projectComponent.SetProjects(msg)
			m.loading = false
		case errMsg:
			m.errorMsg = msg.err.Error()
			m.loading = false
		}
		m.projectComponent, cmd = m.projectComponent.Update(msg)
	case stateLanguageSelection:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c", "q":
				m.quitting = true
				return m, tea.Quit
			case "esc", "b":
				m.state = stateProjectList
				m.languageComponent = nil
				m.errorMsg = ""
				return m, nil
			}
		case language.DownloadCompleteMsg:
			m.state = stateProjectList
			m.languageComponent = nil
			// Update the project component to reflect new download status
			m.projectComponent.UpdateProjectStatus()
			return m, nil
		case language.DownloadProgressMsg:
			// Pass progress to language component
			updatedComponent, cmd := m.languageComponent.Update(msg)
			m.languageComponent = updatedComponent
			return m, cmd
		case language.DownloadErrorMsg:
			m.errorMsg = msg.Error
			// Pass error to language component
			updatedComponent, cmd := m.languageComponent.Update(msg)
			m.languageComponent = updatedComponent
			return m, cmd
		}

		// Update language component
		if m.languageComponent != nil {
			updatedComponent, cmd := m.languageComponent.Update(msg)
			m.languageComponent = updatedComponent
			return m, cmd
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
			m.testComponent.SetProjects(msg)
			m.loading = false
			return m, nil
		case errMsg:
			m.errorMsg = msg.err.Error()
			m.loading = false
			return m, nil
		}

		// Update test component
		updatedComponent, cmd := m.testComponent.Update(msg)
		m.testComponent = updatedComponent
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
		view := asciiArt + "\n"
		view += m.mainMenu.View()
		view += "\n" + m.footer.View(footer.NavigateBinding, footer.EnterBinding, footer.QuitBinding)
		return view
	case stateLogin:
		return m.loginComponent.View()
	case stateProjectList:
		if m.loading {
			return headerStyle.Render("\nLoading projects...")
		}
		helpView := helpStyle.Render(m.help.View(keys) + "  [esc/b] back")
		info := ""
		if m.selectedInfo != "" {
			info = "\n" + m.selectedInfo
		}
		view := fmt.Sprintf("%s\n%s%s", m.projectComponent.View(), helpView, info)
		if m.errorMsg != "" {
			view = fmt.Sprintf("%s\n\n%s", view, errorStyle.Render(m.errorMsg))
		}
		return view
	case stateLanguageSelection:
		if m.languageComponent != nil {
			return m.languageComponent.View()
		}
		return "Loading language selection..."

	case stateTestProject:
		if m.loading {
			return headerStyle.Render("\nLoading projects...")
		}
		return m.testComponent.View()
	}
	return ""
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
