package controller

import (
	"404skill-cli/api"
	"404skill-cli/auth"
	"404skill-cli/config"
	"404skill-cli/downloader"
	"404skill-cli/filesystem"
	"404skill-cli/supabase"
	"404skill-cli/testreport"
	"404skill-cli/testrunner"
	"404skill-cli/tracing"
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
	"fmt"

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

// Controller manages the overall TUI state and coordinates between components
type Controller struct {
	// State management
	stateMachine *state.Machine

	// Key handling
	keyHandler     *keys.Handler
	footerBindings *keys.FooterBindings

	// Tracing integration
	tracer *tracing.TUIIntegration

	// Components
	loginComponent       *login.Component
	projectComponent     *projects.Component
	languageComponent    *language.Component
	testComponent        test.Component
	mainMenu             *menu.Component
	projectNameMenu      *menu.Component
	testProjectNameMenu  *menu.Component
	variantComponent     *variant.Component
	testVariantComponent *variant.Component
	footer               *footer.Component
	help                 help.Model

	// Dependencies
	fileManager    *filesystem.Manager
	configManager  *config.ConfigManager
	client         api.ClientInterface
	downloader     downloader.Downloader
	testRunner     testrunner.TestRunner
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

	// Legacy table support (to be removed)
	table btable.Model
}

// New creates a new TUI controller
func New(client api.ClientInterface, version string, tracer *tracing.TUIIntegration) (*Controller, error) {
	// Track controller initialization
	var initTracker *tracing.TimedOperationTracker
	if tracer != nil {
		initTracker = tracer.TrackProjectOperation("controller_init", "tui_controller")
	}

	// Initialize dependencies
	fileManager := filesystem.NewManager()

	// Create auth provider for dependency injection
	supabaseClient, err := supabase.NewSupabaseClient()
	if err != nil {
		if tracer != nil {
			_ = tracer.TrackError(err, "controller", "supabase_client_creation")
		}
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

	// Track initial state determination
	if tracer != nil {
		stateStr := initialState.String()
		_ = tracer.TrackStateChange("", stateStr, "initial_state_determination")
	}

	// Create state machine
	stateMachine := state.NewMachine(initialState)

	// Create key handler and footer bindings
	keyHandler := keys.NewHandler()
	footerBindings := keys.NewFooterBindings()

	// Create components
	loginComponent := login.New(authProvider, configManager)
	projectComponent := projects.New(client, configManager, fileManager)
	testRunner := testrunner.NewDefaultTestRunner()
	testComponent := test.New(testRunner, configManager, client)
	mainMenu := menu.New([]string{"Download a project", "Test a project"})
	projectNameMenu := menu.New([]string{})
	testProjectNameMenu := menu.New([]string{})
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
		stateMachine:        stateMachine,
		keyHandler:          keyHandler,
		footerBindings:      footerBindings,
		tracer:              tracer,
		loginComponent:      loginComponent,
		projectComponent:    projectComponent,
		testComponent:       testComponent,
		mainMenu:            mainMenu,
		projectNameMenu:     projectNameMenu,
		testProjectNameMenu: testProjectNameMenu,
		footer:              footer,
		help:                help,
		fileManager:         fileManager,
		configManager:       configManager,
		client:              client,
		downloader:          gitDownloader,
		testRunner:          testRunner,
		projectService:      projectService,
		projectUtils:        projectUtils,
		versionChecker:      versionChecker,
		versionInfo:         VersionInfo{CurrentVersion: version},
		table:               btableModel,
	}

	// Complete initialization tracking
	if initTracker != nil {
		_ = initTracker.Complete()
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
		c.cleanup() // Add cleanup before quitting
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
	currentState := c.stateMachine.Current()

	switch currentState {
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
	case state.TestProjectNameMenu:
		return c.handleTestProjectNameMenuState(msg)
	case state.TestProjectVariantMenu:
		return c.handleTestProjectVariantMenuState(msg)
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
			if c.tracer != nil {
				_ = c.tracer.TrackStateChange("refreshing_token", "main_menu", "token_refresh_success")
			}
			return c, c.stateMachine.Transition(state.MainMenu)
		} else {
			if c.tracer != nil {
				_ = c.tracer.TrackError(msg.Error, "controller", "token_refresh")
				_ = c.tracer.TrackStateChange("refreshing_token", "login", "token_refresh_failed")
			}
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

		// Track menu selection
		if c.tracer != nil {
			actionName := "download_project"
			if c.selectedAction == TestProject {
				actionName = "test_project"
			}
			_ = c.tracer.TrackMenuNavigation("main_menu", "select", actionName)
		}

		if c.selectedAction == TestProject {
			if c.tracer != nil {
				_ = c.tracer.TrackStateChange("main_menu", "test_project_name_menu", "test_project_selected")
			}
			return c, tea.Batch(
				c.stateMachine.Transition(state.TestProjectNameMenu),
				c.projectService.FetchProjects(),
			)
		} else {
			if c.tracer != nil {
				_ = c.tracer.TrackStateChange("main_menu", "project_name_menu", "download_project_selected")
			}
			return c, tea.Batch(
				c.stateMachine.Transition(state.ProjectNameMenu),
				c.projectService.FetchProjects(),
			)
		}
	case login.LoginSuccessMsg:
		if c.tracer != nil {
			_ = c.tracer.TrackStateChange("login", "main_menu", "login_success")
		}
		return c, c.stateMachine.Transition(state.MainMenu)
	case login.LoginErrorMsg:
		if c.tracer != nil {
			_ = c.tracer.TrackError(fmt.Errorf("%s", msg.Error), "controller", "login")
			_ = c.tracer.TrackStateChange("main_menu", "login", "login_error")
		}
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
			c.cleanup() // Add cleanup before quitting
			return c, tea.Quit
		}
		// Delegate all other login input to the login component
		updatedComponent, cmd := c.loginComponent.Update(msg)
		c.loginComponent = updatedComponent
		return c, cmd
	case login.LoginSuccessMsg:
		if c.tracer != nil {
			_ = c.tracer.TrackStateChange("login", "main_menu", "login_success")
		}
		return c, c.stateMachine.Transition(state.MainMenu)
	case login.LoginErrorMsg:
		if c.tracer != nil {
			_ = c.tracer.TrackError(fmt.Errorf("%s", msg.Error), "controller", "login")
		}
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

			if c.tracer != nil {
				_ = c.tracer.TrackMenuNavigation("project_name_menu", "select", selectedName)
				_ = c.tracer.TrackStateChange("project_name_menu", "project_variant_menu", "project_selected")
			}

			variants := c.projectUtils.FilterByName(c.projects, c.selectedProjectName)
			c.variantComponent = variant.New(variants, c.downloader, c.configManager, c.fileManager)
			return c, c.stateMachine.Transition(state.ProjectVariantMenu)
		}
		if c.keyHandler.IsBack(msg) {
			if c.tracer != nil {
				_ = c.tracer.TrackStateChange("project_name_menu", "main_menu", "back_key")
			}
			return c, c.stateMachine.Transition(state.MainMenu)
		}
	case domain.ProjectsLoadedMsg:
		if c.tracer != nil {
			projectTracker := c.tracer.TrackAPICall("fetch_projects")
			_ = projectTracker.Complete()
		}
		c.projects = msg.Projects
		c.projectNameMenu.SetItems(c.projectUtils.ExtractUniqueNames(c.projects))
		c.loading = false
		return c, nil
	case domain.ProjectsErrorMsg:
		if c.tracer != nil {
			_ = c.tracer.TrackError(msg.Error, "controller", "fetch_projects")
		}
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
			if c.tracer != nil {
				_ = c.tracer.TrackStateChange("project_variant_menu", "project_name_menu", "back_action")
			}
			return c, c.stateMachine.Transition(state.ProjectNameMenu)
		}

		return c, cmd
	}
	return c, nil
}

func (c *Controller) handleTestProjectNameMenuState(msg tea.Msg) (*Controller, tea.Cmd) {
	// Update test project name menu if projects are loaded
	if len(c.projects) > 0 && len(c.testProjectNameMenu.GetItems()) == 0 {
		// Filter to only show downloaded projects for testing
		downloadedProjects := []api.Project{}
		for _, project := range c.projects {
			if c.configManager.IsProjectDownloaded(project.ID) {
				downloadedProjects = append(downloadedProjects, project)
			}
		}
		c.testProjectNameMenu.SetItems(c.projectUtils.ExtractUniqueNames(downloadedProjects))
	}

	var cmd tea.Cmd
	c.testProjectNameMenu, cmd = c.testProjectNameMenu.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if c.keyHandler.IsEnter(msg) {
			selectedName := c.testProjectNameMenu.GetSelectedItem()
			c.selectedProjectName = selectedName

			if c.tracer != nil {
				_ = c.tracer.TrackMenuNavigation("test_project_name_menu", "select", selectedName)
				_ = c.tracer.TrackStateChange("test_project_name_menu", "test_project_variant_menu", "test_project_selected")
			}

			// Filter to only downloaded projects
			downloadedProjects := []api.Project{}
			for _, project := range c.projects {
				if c.configManager.IsProjectDownloaded(project.ID) {
					downloadedProjects = append(downloadedProjects, project)
				}
			}

			variants := c.projectUtils.FilterByName(downloadedProjects, c.selectedProjectName)
			c.testVariantComponent = variant.NewForTesting(variants, c.testRunner, c.configManager, c.fileManager)
			return c, c.stateMachine.Transition(state.TestProjectVariantMenu)
		}
		if c.keyHandler.IsBack(msg) {
			if c.tracer != nil {
				_ = c.tracer.TrackStateChange("test_project_name_menu", "main_menu", "back_key")
			}
			return c, c.stateMachine.Transition(state.MainMenu)
		}
	case domain.ProjectsLoadedMsg:
		if c.tracer != nil {
			projectTracker := c.tracer.TrackAPICall("fetch_projects_for_testing")
			_ = projectTracker.Complete()
		}
		c.projects = msg.Projects
		// Filter to only show downloaded projects for testing
		downloadedProjects := []api.Project{}
		for _, project := range c.projects {
			if c.configManager.IsProjectDownloaded(project.ID) {
				downloadedProjects = append(downloadedProjects, project)
			}
		}
		c.testProjectNameMenu.SetItems(c.projectUtils.ExtractUniqueNames(downloadedProjects))
		c.loading = false
		return c, nil
	case domain.ProjectsErrorMsg:
		if c.tracer != nil {
			_ = c.tracer.TrackError(msg.Error, "controller", "fetch_projects_for_testing")
		}
		c.errorMsg = msg.Error.Error()
		c.loading = false
		return c, nil
	}

	return c, cmd
}

func (c *Controller) handleTestProjectVariantMenuState(msg tea.Msg) (*Controller, tea.Cmd) {
	if c.testVariantComponent != nil {
		updated, cmd := c.testVariantComponent.Update(msg)
		c.testVariantComponent = updated

		// Handle test completion - navigate to test results
		switch msg := msg.(type) {
		case variant.TestCompleteMsg:
			if c.tracer != nil {
				_ = c.tracer.TrackStateChange("test_project_variant_menu", "test_project", "test_completed")
			}
			// Convert the test result and show in test component
			// We need to send the test result to the test component
			return c, tea.Batch(
				c.stateMachine.Transition(state.TestProject),
				func() tea.Msg {
					// Convert variant.TestCompleteMsg to test.TestCompleteMsg
					testResult, ok := msg.Result.(*testreport.ParseResult)
					if !ok {
						return test.TestErrorMsg{Error: "Invalid test result format"}
					}
					return test.TestCompleteMsg{
						Project: &testrunner.Project{
							ID:       msg.Variant.ID,
							Name:     msg.Variant.Name,
							Language: msg.Variant.Language,
						},
						Result: testResult,
					}
				},
			)
		case variant.TestErrorMsg:
			if c.tracer != nil {
				_ = c.tracer.TrackError(fmt.Errorf("%s", msg.Error), "controller", "test_execution")
			}
			c.errorMsg = msg.Error
			return c, nil
		}

		if _, ok := msg.(variant.BackMsg); ok {
			if c.tracer != nil {
				_ = c.tracer.TrackStateChange("test_project_variant_menu", "test_project_name_menu", "back_action")
			}
			return c, c.stateMachine.Transition(state.TestProjectNameMenu)
		}

		return c, cmd
	}
	return c, nil
}

func (c *Controller) handleTestProjectState(msg tea.Msg) (*Controller, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if c.keyHandler.IsBack(msg) {
			if c.tracer != nil {
				_ = c.tracer.TrackStateChange("test_project", "main_menu", "back_key")
			}
			return c, c.stateMachine.Transition(state.MainMenu)
		}
	case domain.ProjectsLoadedMsg:
		c.projects = msg.Projects
		c.loading = false
		return c, nil
	case domain.ProjectsErrorMsg:
		if c.tracer != nil {
			_ = c.tracer.TrackError(msg.Error, "controller", "test_project_state")
		}
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
	case state.TestProjectNameMenu:
		return c.renderTestProjectNameMenu()
	case state.TestProjectVariantMenu:
		return c.renderTestProjectVariantMenu()
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

// cleanup properly shuts down background processes and tickers
func (c *Controller) cleanup() {
	// Track application shutdown
	if c.tracer != nil {
		_ = c.tracer.TrackStateChange(c.stateMachine.Current().String(), "application_exit", "user_quit")
	}
}
