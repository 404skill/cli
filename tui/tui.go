package tui

import (
	"404skill-cli/api"
	"404skill-cli/tui/controller"

	tea "github.com/charmbracelet/bubbletea"
)

// Model wraps the controller for the Bubble Tea framework
type Model struct {
	controller *controller.Controller
}

// InitialModel creates a new TUI model with the given API client and version
func InitialModel(client api.ClientInterface, version string) (Model, error) {
	ctrl, err := controller.New(client, version)
	if err != nil {
		return Model{}, err
	}

	return Model{
		controller: ctrl,
	}, nil
}

// Init initializes the model and returns initial commands
func (m Model) Init() tea.Cmd {
	return m.controller.Init()
}

// Update handles incoming messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	updatedController, cmd := m.controller.Update(msg)
	m.controller = updatedController
	return m, cmd
}

// View renders the current state of the model
func (m Model) View() string {
	return m.controller.View()
}

// IsQuitting returns true if the application is quitting
func (m Model) IsQuitting() bool {
	return m.controller.IsQuitting()
}
