package state

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// State represents the current state of the TUI application
type State int

const (
	// RefreshingToken - Application is refreshing the user's authentication token/session
	RefreshingToken State = iota

	// MainMenu - Main menu displaying "Download a project" and "Test a project" options
	MainMenu

	// Login - User authentication screen for entering credentials
	Login

	// ProjectNameMenu - Menu showing unique project names for selection
	ProjectNameMenu

	// ProjectVariantMenu - Menu showing project variants (by technology stack) for selected project
	ProjectVariantMenu

	// TestProject - Test project functionality screen with project selection and test execution
	TestProject
)

// String returns a human-readable representation of the state
func (s State) String() string {
	switch s {
	case RefreshingToken:
		return "RefreshingToken"
	case MainMenu:
		return "MainMenu"
	case Login:
		return "Login"
	case ProjectNameMenu:
		return "ProjectNameMenu"
	case ProjectVariantMenu:
		return "ProjectVariantMenu"
	case TestProject:
		return "TestProject"
	default:
		return fmt.Sprintf("Unknown(%d)", int(s))
	}
}

// IsValid checks if the state is a valid state
func (s State) IsValid() bool {
	return s >= RefreshingToken && s <= TestProject
}

// Transition represents a state transition
type Transition struct {
	From State
	To   State
}

// String returns a human-readable representation of the transition
func (t Transition) String() string {
	return fmt.Sprintf("%s -> %s", t.From, t.To)
}

// Machine manages state transitions and validation
type Machine struct {
	current State
	history []State
}

// NewMachine creates a new state machine with the given initial state
func NewMachine(initial State) *Machine {
	return &Machine{
		current: initial,
		history: []State{initial},
	}
}

// Current returns the current state
func (m *Machine) Current() State {
	return m.current
}

// Transition transitions to a new state
func (m *Machine) Transition(to State) tea.Cmd {
	if !to.IsValid() {
		return func() tea.Msg {
			return ErrorMsg{
				Error: fmt.Errorf("invalid state transition to %s", to),
			}
		}
	}

	transition := Transition{From: m.current, To: to}
	m.current = to
	m.history = append(m.history, to)

	return func() tea.Msg {
		return TransitionMsg{Transition: transition}
	}
}

// CanGoBack returns true if there's a previous state to go back to
func (m *Machine) CanGoBack() bool {
	return len(m.history) > 1
}

// GoBack transitions to the previous state
func (m *Machine) GoBack() tea.Cmd {
	if !m.CanGoBack() {
		return nil
	}

	// Remove current state and get previous
	m.history = m.history[:len(m.history)-1]
	previous := m.history[len(m.history)-1]

	transition := Transition{From: m.current, To: previous}
	m.current = previous

	return func() tea.Msg {
		return TransitionMsg{Transition: transition}
	}
}

// History returns a copy of the state history
func (m *Machine) History() []State {
	history := make([]State, len(m.history))
	copy(history, m.history)
	return history
}

// Reset resets the state machine to the given state
func (m *Machine) Reset(initial State) {
	m.current = initial
	m.history = []State{initial}
}

// Messages for state machine events
type (
	// TransitionMsg is sent when a state transition occurs
	TransitionMsg struct {
		Transition Transition
	}

	// ErrorMsg is sent when a state machine error occurs
	ErrorMsg struct {
		Error error
	}
)
