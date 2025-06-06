package tui

import (
	"404skill-cli/tui/login"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
)

// State represents the current state of the application
type State int

const (
	StateRefreshingToken State = iota
	StateMainMenu
	StateLogin
	StateProjectList
	StateLanguageSelection
	StateConfirmRedownload
	StateTestProject
)

// MainMenuChoice represents the available choices in the main menu
type MainMenuChoice int

const (
	ChoiceInit MainMenuChoice = iota
	ChoiceTest
)

// Model represents the main application model
type Model struct {
	// State
	state           State
	mainMenuIndex   int
	mainMenuChoices []string
	selectedAction  MainMenuChoice

	// Components
	loginComponent    *login.Component
	projectComponent  *ProjectComponent
	languageComponent *LanguageComponent
	testComponent     *TestComponent
	help              help.Model

	// State
	ready    bool
	quitting bool
	errorMsg string
}

// Component represents a UI component that can be updated and rendered
type Component interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (Component, tea.Cmd)
	View() string
}

// Operation represents a command that can be executed
type Operation interface {
	Execute() tea.Cmd
}

// ProgressObserver handles progress updates
type ProgressObserver interface {
	OnProgress(progress float64)
	OnComplete()
	OnError(err error)
}

// FileManager handles file system operations
type FileManager interface {
	OpenFileExplorer(path string) error
	CreateDirectory(path string) error
	RemoveDirectory(path string) error
	DirectoryExists(path string) bool
}

// ConfigManager handles configuration operations
type ConfigManager interface {
	HasCredentials() bool
	GetDownloadedProjects() map[string]bool
	UpdateDownloadedProject(projectID string) error
	IsProjectDownloaded(projectID string) bool
}
