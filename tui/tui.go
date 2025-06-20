package tui

import (
	"404skill-cli/api"
	"404skill-cli/tracing"
	"404skill-cli/tui/controller"

	tea "github.com/charmbracelet/bubbletea"
)

// Model wraps the controller for the Bubble Tea framework
type Model struct {
	controller *controller.Controller
	tracer     *tracing.TUIIntegration
}

// InitialModel creates a new TUI model with the given API client and version
func InitialModel(client api.ClientInterface, version string) (Model, error) {
	// Get global tracing manager and create TUI integration
	var tuiTracer *tracing.TUIIntegration
	if manager := tracing.GetGlobalManager(); manager != nil {
		tuiTracer = tracing.NewTUIIntegration(manager)
	}

	ctrl, err := controller.New(client, version, tuiTracer)
	if err != nil {
		if tuiTracer != nil {
			_ = tuiTracer.TrackError(err, "tui", "initialization")
		}
		return Model{}, err
	}

	return Model{
		controller: ctrl,
		tracer:     tuiTracer,
	}, nil
}

// Init initializes the model and returns initial commands
func (m Model) Init() tea.Cmd {
	if m.tracer != nil {
		_ = m.tracer.TrackStateChange("", "tui_init", "bubble_tea_init")
	}
	return m.controller.Init()
}

// Update handles incoming messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Track key presses if we have a tracer
	if m.tracer != nil {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			currentState := m.controller.CurrentState().String()
			_ = m.tracer.TrackKeyMsg(keyMsg, currentState)
		}
	}

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
