package controller

import (
	"404skill-cli/api"
	"404skill-cli/auth"
	"404skill-cli/config"
	"404skill-cli/downloader"
	"404skill-cli/filesystem"
	"404skill-cli/supabase"
	"404skill-cli/testrunner"
	"404skill-cli/tui/components/footer"
	"404skill-cli/tui/components/menu"
	"404skill-cli/tui/domain"
	"404skill-cli/tui/keys"
	"404skill-cli/tui/language"
	"404skill-cli/tui/login"
	"404skill-cli/tui/projects"
	"404skill-cli/tui/state"
	"404skill-cli/tui/test"
	"404skill-cli/tui/variant"
	"time"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	btable "github.com/evertras/bubble-table/table"
)

// MainMenuAction represents the main menu choices
type MainMenuAction int

const (
	DownloadProject MainMenuAction = iota
	TestProject
)

// Controller orchestrates the TUI application
type Controller struct {
	// State management
	stateMachine *state.Machine

	// Key handling
	keyHandler     *keys.Handler
	footerBindings *keys.FooterBindings

	// Components
	loginComponent    *login.Component
	projectComponent  *projects.Component
	languageComponent *language.Component
	testComponent     test.Component
	mainMenu          *menu.Component
	projectNameMenu   *menu.Component
	variantComponent  *variant.Component
	footer            *footer.Component
	help              help.Model

	// Dependencies
	fileManager    *filesystem.Manager
	configManager  *config.ConfigManager
	client         api.ClientInterface
	downloader     downloader.Downloader
	projectService *domain.ProjectService
	projectUtils   *domain.ProjectUtils
	versionChecker *VersionChecker

	// Application state
	projects            []api.Project
	selectedProjectName string
	selectedAction      MainMenuAction
	loading             bool
	errorMsg            string
	quitting            bool
	versionInfo         VersionInfo
	versionTicker       *time.Ticker

	// Legacy table support (to be removed)
	table btable.Model
}

// New creates a new TUI controller
func New(client api.ClientInterface, version string) (*Controller, error) {
	// Initialize dependencies
	fileManager := filesystem.NewManager()

	// Create auth provider for dependency injection
	supabaseClient, err := supabase.NewSupabaseClient()
	if err != nil {
		// Handle error appropriately - for now we'll continue with nil
		// In production, you might want to handle this differently
	}
	authProvider := auth.NewSupabaseAuth(supabaseClient)

	// Create a basic config writer that doesn't depend on auth service
	configWriter := config.SimpleConfigWriter{}

	// Create auth service with dependencies
	authService := auth.NewAuthService(authProvider, &configWriter)

	// Create config manager with auth service dependency
	configManager := config.NewConfigManager(authService)

	// Determine initial state
	initialState := state.Login
	if configManager.HasCredentials() {
		initialState = state.RefreshingToken
	}

	// Create state machine
	stateMachine := state.NewMachine(initialState)

	// Create key handler and footer bindings
	keyHandler := keys.NewHandler()
	footerBindings := keys.NewFooterBindings()

	// Create components
	loginComponent := login.New(authProvider, configManager)
	projectComponent := projects.New(client, configManager, fileManager)
	testComponent := test.New(testrunner.NewDefaultTestRunner(), configManager, client)
	mainMenu := menu.New([]string{"Download a project", "Test a project"})
	projectNameMenu := menu.New([]string{})
	footer := footer.New()
	help := help.New()

	// Create downloader
	gitDownloader := downloader.NewGitDownloader(fileManager, configManager, client)

	// Create domain services
	projectService := domain.NewProjectService(client)
	projectUtils := domain.NewProjectUtils()

	// Create version checker
	versionChecker := NewVersionChecker(version)

	// Create legacy table (to be removed)
	rows := []btable.Row{}
	btableModel := btable.New(projectUtils.CreateTableColumns()).WithRows(rows)

	controller := &Controller{
		stateMachine:     stateMachine,
		keyHandler:       keyHandler,
		footerBindings:   footerBindings,
		loginComponent:   loginComponent,
		projectComponent: projectComponent,
		testComponent:    testComponent,
		mainMenu:         mainMenu,
		projectNameMenu:  projectNameMenu,
		footer:           footer,
		help:             help,
		fileManager:      fileManager,
		configManager:    configManager,
		client:           client,
		downloader:       gitDownloader,
		projectService:   projectService,
		projectUtils:     projectUtils,
		versionChecker:   versionChecker,
		versionInfo:      VersionInfo{CurrentVersion: version},
		table:            btableModel,
	}

	return controller, nil
}

// Init initializes the controller and returns initial commands
func (c *Controller) Init() tea.Cmd {
	commands := []tea.Cmd{
		c.checkVersionCmd(),
		c.versionTickerCmd(),
	}

	if c.configManager.HasCredentials() {
		commands = append(commands, c.refreshTokenCmd())
	}

	return tea.Batch(commands...)
}

// Update handles incoming messages and updates the controller state
func (c *Controller) Update(msg tea.Msg) (*Controller, tea.Cmd) {
	// Handle global quit
	if keyMsg, ok := msg.(tea.KeyMsg); ok && c.keyHandler.IsQuit(keyMsg) {
		c.quitting = true
		return c, tea.Quit
	}

	// Handle global messages
	switch msg := msg.(type) {
	case VersionCheckMsg:
		c.versionInfo = msg.Info
		return c, nil
	case VersionTickerMsg:
		return c, c.checkVersionCmd()
	case state.ErrorMsg:
		c.errorMsg = msg.Error.Error()
		return c, nil
	}

	// Delegate to state-specific handlers
	return c.handleStateUpdate(msg)
}

// handleStateUpdate delegates message handling based on current state
func (c *Controller) handleStateUpdate(msg tea.Msg) (*Controller, tea.Cmd) {
	switch c.stateMachine.Current() {
	case state.RefreshingToken:
		return c.handleRefreshingTokenState(msg)
	case state.MainMenu:
		return c.handleMainMenuState(msg)
	case state.Login:
		return c.handleLoginState(msg)
	case state.ProjectNameMenu:
		return c.handleProjectNameMenuState(msg)
	case state.ProjectVariantMenu:
		return c.handleProjectVariantMenuState(msg)
	case state.TestProject:
		return c.handleTestProjectState(msg)
	default:
		return c, nil
	}
}

// State-specific handlers
func (c *Controller) handleRefreshingTokenState(msg tea.Msg) (*Controller, tea.Cmd) {
	switch msg := msg.(type) {
	case TokenRefreshMsg:
		if msg.Error == nil {
			return c, c.stateMachine.Transition(state.MainMenu)
		} else {
			c.loginComponent.SetError("Session expired. Please log in again.")
			return c, c.stateMachine.Transition(state.Login)
		}
	case VersionCheckMsg:
		c.versionInfo = msg.Info
		return c, nil
	}
	// Block all other input during token refresh
	return c, nil
}

func (c *Controller) handleMainMenuState(msg tea.Msg) (*Controller, tea.Cmd) {
	// Update main menu component
	var menuCmd tea.Cmd
	c.mainMenu, menuCmd = c.mainMenu.Update(msg)

	switch msg := msg.(type) {
	case menu.MenuSelectMsg:
		c.selectedAction = MainMenuAction(msg.SelectedIndex)
		c.loading = true

		if c.selectedAction == TestProject {
			return c, tea.Batch(
				c.stateMachine.Transition(state.TestProject),
				c.projectService.FetchProjects(),
			)
		} else {
			return c, tea.Batch(
				c.stateMachine.Transition(state.ProjectNameMenu),
				c.projectService.FetchProjects(),
			)
		}
	case login.LoginSuccessMsg:
		return c, c.stateMachine.Transition(state.MainMenu)
	case login.LoginErrorMsg:
		c.loginComponent.SetError(msg.Error)
		return c, c.stateMachine.Transition(state.Login)
	}

	if menuCmd != nil {
		return c, menuCmd
	}
	return c, nil
}

func (c *Controller) handleLoginState(msg tea.Msg) (*Controller, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if c.keyHandler.IsQuit(msg) {
			c.quitting = true
			return c, tea.Quit
		}
		// Delegate all other login input to the login component
		updatedComponent, cmd := c.loginComponent.Update(msg)
		c.loginComponent = updatedComponent
		return c, cmd
	case login.LoginSuccessMsg:
		return c, c.stateMachine.Transition(state.MainMenu)
	case login.LoginErrorMsg:
		c.loginComponent.SetError(msg.Error)
		return c, nil
	}
	return c, nil
}

func (c *Controller) handleProjectNameMenuState(msg tea.Msg) (*Controller, tea.Cmd) {
	// Update project name menu if projects are loaded
	if len(c.projects) > 0 && len(c.projectNameMenu.GetItems()) == 0 {
		c.projectNameMenu.SetItems(c.projectUtils.ExtractUniqueNames(c.projects))
	}

	var cmd tea.Cmd
	c.projectNameMenu, cmd = c.projectNameMenu.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if c.keyHandler.IsEnter(msg) {
			selectedName := c.projectNameMenu.GetSelectedItem()
			c.selectedProjectName = selectedName
			variants := c.projectUtils.FilterByName(c.projects, c.selectedProjectName)
			c.variantComponent = variant.New(variants, c.downloader, c.configManager, c.fileManager)
			return c, c.stateMachine.Transition(state.ProjectVariantMenu)
		}
		if c.keyHandler.IsBack(msg) {
			return c, c.stateMachine.Transition(state.MainMenu)
		}
	case domain.ProjectsLoadedMsg:
		c.projects = msg.Projects
		c.projectNameMenu.SetItems(c.projectUtils.ExtractUniqueNames(c.projects))
		c.loading = false
		return c, nil
	case domain.ProjectsErrorMsg:
		c.errorMsg = msg.Error.Error()
		c.loading = false
		return c, nil
	}

	return c, cmd
}

func (c *Controller) handleProjectVariantMenuState(msg tea.Msg) (*Controller, tea.Cmd) {
	if c.variantComponent != nil {
		updated, cmd := c.variantComponent.Update(msg)
		c.variantComponent = updated

		if _, ok := msg.(variant.BackMsg); ok {
			return c, c.stateMachine.Transition(state.ProjectNameMenu)
		}

		return c, cmd
	}
	return c, nil
}

func (c *Controller) handleTestProjectState(msg tea.Msg) (*Controller, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if c.keyHandler.IsBack(msg) {
			return c, c.stateMachine.Transition(state.MainMenu)
		}
	case domain.ProjectsLoadedMsg:
		c.projects = msg.Projects
		c.loading = false
		return c, nil
	case domain.ProjectsErrorMsg:
		c.errorMsg = msg.Error.Error()
		c.loading = false
		return c, nil
	}

	// Delegate to test component
	updatedComponent, cmd := c.testComponent.Update(msg)
	c.testComponent = updatedComponent
	return c, cmd
}

// View renders the current state
func (c *Controller) View() string {
	if c.quitting {
		return c.renderQuitting()
	}

	switch c.stateMachine.Current() {
	case state.RefreshingToken:
		return c.renderRefreshingToken()
	case state.MainMenu:
		return c.renderMainMenu()
	case state.Login:
		return c.renderLogin()
	case state.ProjectNameMenu:
		return c.renderProjectNameMenu()
	case state.ProjectVariantMenu:
		return c.renderProjectVariantMenu()
	case state.TestProject:
		return c.renderTestProject()
	default:
		return "Unknown state"
	}
}

// Getters for accessing controller state
func (c *Controller) IsQuitting() bool {
	return c.quitting
}

func (c *Controller) CurrentState() state.State {
	return c.stateMachine.Current()
}

func (c *Controller) GetProjects() []api.Project {
	return c.projects
}

func (c *Controller) IsLoading() bool {
	return c.loading
}

func (c *Controller) GetErrorMsg() string {
	return c.errorMsg
}

func (c *Controller) GetVersionInfo() VersionInfo {
	return c.versionInfo
}
