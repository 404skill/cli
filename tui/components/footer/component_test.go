package footer

import (
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	// Act
	component := New()

	// Assert
	if component == nil {
		t.Error("Expected component to be created")
	}
	// Test that style is initialized by checking if it's not the zero value
	// We can't compare styles directly, so we test behavior instead
	testResult := component.View(KeyBinding{Key: "test", Description: "test"})
	if testResult == "" {
		t.Error("Expected styled output from component")
	}
}

func TestKeyBinding_Format(t *testing.T) {
	tests := []struct {
		name     string
		binding  KeyBinding
		expected string
	}{
		{
			name:     "valid binding",
			binding:  KeyBinding{Key: "q", Description: "quit"},
			expected: "[q] quit",
		},
		{
			name:     "empty key",
			binding:  KeyBinding{Key: "", Description: "quit"},
			expected: "",
		},
		{
			name:     "empty description",
			binding:  KeyBinding{Key: "q", Description: ""},
			expected: "",
		},
		{
			name:     "both empty",
			binding:  KeyBinding{Key: "", Description: ""},
			expected: "",
		},
		{
			name:     "complex key combination",
			binding:  KeyBinding{Key: "↑/↓ or k/j", Description: "move"},
			expected: "[↑/↓ or k/j] move",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			result := tt.binding.Format()

			// Assert
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestComponent_View_EmptyBindings(t *testing.T) {
	// Arrange
	component := New()

	// Act
	result := component.View()

	// Assert
	if result != "" {
		t.Errorf("Expected empty string for no bindings, got '%s'", result)
	}
}

func TestComponent_View_SingleBinding(t *testing.T) {
	// Arrange
	component := New()
	binding := KeyBinding{Key: "q", Description: "quit"}

	// Act
	result := component.View(binding)

	// Assert
	expected := "[q] quit"
	if !strings.Contains(result, expected) {
		t.Errorf("Expected result to contain '%s', got '%s'", expected, result)
	}
}

func TestComponent_View_MultipleBindings(t *testing.T) {
	// Arrange
	component := New()
	bindings := []KeyBinding{
		{Key: "q", Description: "quit"},
		{Key: "enter", Description: "select"},
		{Key: "esc/b", Description: "back"},
	}

	// Act
	result := component.View(bindings...)

	// Assert
	expectedParts := []string{"[q] quit", "[enter] select", "[esc/b] back"}
	for _, part := range expectedParts {
		if !strings.Contains(result, part) {
			t.Errorf("Expected result to contain '%s', got '%s'", part, result)
		}
	}

	// Should contain separators between bindings
	if !strings.Contains(result, "  ") {
		t.Error("Expected bindings to be separated by two spaces")
	}
}

func TestComponent_View_SkipsInvalidBindings(t *testing.T) {
	// Arrange
	component := New()
	bindings := []KeyBinding{
		{Key: "q", Description: "quit"},     // Valid
		{Key: "", Description: "invalid"},   // Invalid - empty key
		{Key: "enter", Description: ""},     // Invalid - empty description
		{Key: "esc/b", Description: "back"}, // Valid
	}

	// Act
	result := component.View(bindings...)

	// Assert
	// Should contain valid bindings
	if !strings.Contains(result, "[q] quit") {
		t.Error("Expected result to contain valid binding '[q] quit'")
	}
	if !strings.Contains(result, "[esc/b] back") {
		t.Error("Expected result to contain valid binding '[esc/b] back'")
	}

	// Should not contain invalid bindings (empty strings)
	parts := strings.Split(result, "  ")
	validParts := 0
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			validParts++
		}
	}
	if validParts != 2 {
		t.Errorf("Expected 2 valid parts, got %d", validParts)
	}
}

func TestPredefinedBindings(t *testing.T) {
	tests := []struct {
		name    string
		binding KeyBinding
		key     string
		desc    string
	}{
		{
			name:    "quit binding",
			binding: QuitBinding,
			key:     "q",
			desc:    "quit",
		},
		{
			name:    "back binding",
			binding: BackBinding,
			key:     "esc/b",
			desc:    "back",
		},
		{
			name:    "enter binding",
			binding: EnterBinding,
			key:     "enter",
			desc:    "select",
		},
		{
			name:    "confirm binding",
			binding: ConfirmBinding,
			key:     "enter",
			desc:    "confirm",
		},
		{
			name:    "submit binding",
			binding: SubmitBinding,
			key:     "enter",
			desc:    "submit",
		},
		{
			name:    "tab binding",
			binding: TabBinding,
			key:     "tab",
			desc:    "switch",
		},
		{
			name:    "navigate binding",
			binding: NavigateBinding,
			key:     "↑/↓ or k/j",
			desc:    "move",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Assert
			if tt.binding.Key != tt.key {
				t.Errorf("Expected key '%s', got '%s'", tt.key, tt.binding.Key)
			}
			if tt.binding.Description != tt.desc {
				t.Errorf("Expected description '%s', got '%s'", tt.desc, tt.binding.Description)
			}
		})
	}
}

func TestComponent_View_CommonUseCases(t *testing.T) {
	component := New()

	tests := []struct {
		name     string
		bindings []KeyBinding
		contains []string
	}{
		{
			name:     "main menu footer",
			bindings: []KeyBinding{NavigateBinding, EnterBinding, QuitBinding},
			contains: []string{"[↑/↓ or k/j] move", "[enter] select", "[q] quit"},
		},
		{
			name:     "sub-page footer",
			bindings: []KeyBinding{NavigateBinding, EnterBinding, BackBinding, QuitBinding},
			contains: []string{"[↑/↓ or k/j] move", "[enter] select", "[esc/b] back", "[q] quit"},
		},
		{
			name:     "login footer",
			bindings: []KeyBinding{TabBinding, SubmitBinding, QuitBinding},
			contains: []string{"[tab] switch", "[enter] submit", "[q] quit"},
		},
		{
			name:     "confirmation footer",
			bindings: []KeyBinding{NavigateBinding, ConfirmBinding, BackBinding, QuitBinding},
			contains: []string{"[↑/↓ or k/j] move", "[enter] confirm", "[esc/b] back", "[q] quit"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			result := component.View(tt.bindings...)

			// Assert
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected result to contain '%s', got '%s'", expected, result)
				}
			}
		})
	}
}
