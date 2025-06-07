package menu

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestNew(t *testing.T) {
	items := []string{"Item 1", "Item 2", "Item 3"}
	menu := New(items)

	if menu == nil {
		t.Fatal("Expected menu to be created")
	}
	if len(menu.items) != 3 {
		t.Errorf("Expected 3 items, got %d", len(menu.items))
	}
	if menu.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex to be 0, got %d", menu.selectedIndex)
	}
	if menu.GetSelectedItem() != "Item 1" {
		t.Errorf("Expected selected item to be 'Item 1', got '%s'", menu.GetSelectedItem())
	}
}

func TestSetItems(t *testing.T) {
	menu := New([]string{"Old Item"})
	newItems := []string{"New Item 1", "New Item 2"}

	menu.SetItems(newItems)

	if len(menu.items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(menu.items))
	}
	if menu.items[0] != "New Item 1" {
		t.Errorf("Expected first item to be 'New Item 1', got '%s'", menu.items[0])
	}
	if menu.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex to be reset to 0, got %d", menu.selectedIndex)
	}
}

func TestSetItemsResetsSelection(t *testing.T) {
	menu := New([]string{"Item 1", "Item 2", "Item 3"})
	menu.SetSelectedIndex(2)

	// Set fewer items
	menu.SetItems([]string{"New Item"})

	if menu.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex to be reset to 0 when out of bounds, got %d", menu.selectedIndex)
	}
}

func TestGetItems(t *testing.T) {
	items := []string{"Item 1", "Item 2"}
	menu := New(items)

	retrieved := menu.GetItems()
	if len(retrieved) != 2 {
		t.Errorf("Expected 2 items, got %d", len(retrieved))
	}
	if retrieved[0] != "Item 1" {
		t.Errorf("Expected first item to be 'Item 1', got '%s'", retrieved[0])
	}
}

func TestSetSelectedIndex(t *testing.T) {
	menu := New([]string{"Item 1", "Item 2", "Item 3"})

	// Valid index
	menu.SetSelectedIndex(2)
	if menu.selectedIndex != 2 {
		t.Errorf("Expected selectedIndex to be 2, got %d", menu.selectedIndex)
	}

	// Invalid index (too high)
	menu.SetSelectedIndex(5)
	if menu.selectedIndex != 2 {
		t.Errorf("Expected selectedIndex to remain 2 for invalid index, got %d", menu.selectedIndex)
	}

	// Invalid index (negative)
	menu.SetSelectedIndex(-1)
	if menu.selectedIndex != 2 {
		t.Errorf("Expected selectedIndex to remain 2 for negative index, got %d", menu.selectedIndex)
	}
}

func TestGetSelectedIndex(t *testing.T) {
	menu := New([]string{"Item 1", "Item 2", "Item 3"})

	if menu.GetSelectedIndex() != 0 {
		t.Errorf("Expected initial selectedIndex to be 0, got %d", menu.GetSelectedIndex())
	}

	menu.SetSelectedIndex(1)
	if menu.GetSelectedIndex() != 1 {
		t.Errorf("Expected selectedIndex to be 1, got %d", menu.GetSelectedIndex())
	}
}

func TestGetSelectedItem(t *testing.T) {
	menu := New([]string{"Item 1", "Item 2", "Item 3"})

	if menu.GetSelectedItem() != "Item 1" {
		t.Errorf("Expected selected item to be 'Item 1', got '%s'", menu.GetSelectedItem())
	}

	menu.SetSelectedIndex(2)
	if menu.GetSelectedItem() != "Item 3" {
		t.Errorf("Expected selected item to be 'Item 3', got '%s'", menu.GetSelectedItem())
	}
}

func TestGetSelectedItemEmptyMenu(t *testing.T) {
	menu := New([]string{})

	if menu.GetSelectedItem() != "" {
		t.Errorf("Expected empty string for empty menu, got '%s'", menu.GetSelectedItem())
	}
}

func TestSetStyles(t *testing.T) {
	menu := New([]string{"Item 1"})

	customStyles := Styles{
		ItemStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff")),
		SelectedStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("#000000")),
		Cursor:         "* ",
		SelectedCursor: ">> ",
	}

	menu.SetStyles(customStyles)

	if menu.styles.Cursor != "* " {
		t.Errorf("Expected cursor to be '* ', got '%s'", menu.styles.Cursor)
	}
	if menu.styles.SelectedCursor != ">> " {
		t.Errorf("Expected selected cursor to be '>> ', got '%s'", menu.styles.SelectedCursor)
	}
}

func TestUpdateNavigationUp(t *testing.T) {
	menu := New([]string{"Item 1", "Item 2", "Item 3"})
	menu.SetSelectedIndex(1)

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}
	newMenu, _ := menu.Update(keyMsg)

	if newMenu.GetSelectedIndex() != 0 {
		t.Errorf("Expected selectedIndex to be 0 after up navigation, got %d", newMenu.GetSelectedIndex())
	}
}

func TestUpdateNavigationDown(t *testing.T) {
	menu := New([]string{"Item 1", "Item 2", "Item 3"})

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
	newMenu, _ := menu.Update(keyMsg)

	if newMenu.GetSelectedIndex() != 1 {
		t.Errorf("Expected selectedIndex to be 1 after down navigation, got %d", newMenu.GetSelectedIndex())
	}
}

func TestUpdateNavigationWrapAround(t *testing.T) {
	menu := New([]string{"Item 1", "Item 2", "Item 3"})

	// Test wrapping up from first item
	keyMsg := tea.KeyMsg{Type: tea.KeyUp}
	newMenu, _ := menu.Update(keyMsg)
	if newMenu.GetSelectedIndex() != 2 {
		t.Errorf("Expected selectedIndex to wrap to 2 when going up from 0, got %d", newMenu.GetSelectedIndex())
	}

	// Test wrapping down from last item
	menu.SetSelectedIndex(2)
	keyMsg = tea.KeyMsg{Type: tea.KeyDown}
	newMenu, _ = menu.Update(keyMsg)
	if newMenu.GetSelectedIndex() != 0 {
		t.Errorf("Expected selectedIndex to wrap to 0 when going down from 2, got %d", newMenu.GetSelectedIndex())
	}
}

func TestUpdateEnterSelection(t *testing.T) {
	menu := New([]string{"Item 1", "Item 2", "Item 3"})
	menu.SetSelectedIndex(1)

	keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
	newMenu, cmd := menu.Update(keyMsg)

	if newMenu.GetSelectedIndex() != 1 {
		t.Errorf("Expected selectedIndex to remain 1, got %d", newMenu.GetSelectedIndex())
	}

	if cmd == nil {
		t.Fatal("Expected command to be returned for enter key")
	}

	// Execute the command to get the message
	msg := cmd()
	selectMsg, ok := msg.(MenuSelectMsg)
	if !ok {
		t.Fatal("Expected MenuSelectMsg")
	}

	if selectMsg.SelectedIndex != 1 {
		t.Errorf("Expected selected index to be 1, got %d", selectMsg.SelectedIndex)
	}
	if selectMsg.SelectedItem != "Item 2" {
		t.Errorf("Expected selected item to be 'Item 2', got '%s'", selectMsg.SelectedItem)
	}
}

func TestUpdateEmptyMenu(t *testing.T) {
	menu := New([]string{})

	keyMsg := tea.KeyMsg{Type: tea.KeyDown}
	newMenu, cmd := menu.Update(keyMsg)

	if newMenu == nil {
		t.Fatal("Expected menu to be returned")
	}
	if cmd != nil {
		t.Error("Expected no command for empty menu")
	}
}

func TestView(t *testing.T) {
	menu := New([]string{"Item 1", "Item 2", "Item 3"})

	view := menu.View()

	if !strings.Contains(view, "Item 1") {
		t.Error("Expected view to contain 'Item 1'")
	}
	if !strings.Contains(view, "Item 2") {
		t.Error("Expected view to contain 'Item 2'")
	}
	if !strings.Contains(view, "Item 3") {
		t.Error("Expected view to contain 'Item 3'")
	}

	// Check that first item is selected (should have different formatting)
	lines := strings.Split(view, "\n")
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines in view, got %d", len(lines))
	}
}

func TestViewSelection(t *testing.T) {
	menu := New([]string{"Item 1", "Item 2"})
	menu.SetSelectedIndex(1)

	view := menu.View()
	lines := strings.Split(view, "\n")

	// First line should have normal cursor
	if !strings.HasPrefix(lines[0], "  ") {
		t.Error("Expected first line to start with normal cursor '  '")
	}

	// Second line should have selected cursor
	if !strings.HasPrefix(lines[1], "> ") {
		t.Error("Expected second line to start with selected cursor '> '")
	}
}

func TestViewEmptyMenu(t *testing.T) {
	menu := New([]string{})

	view := menu.View()

	if view != "" {
		t.Errorf("Expected empty view for empty menu, got '%s'", view)
	}
}

func TestIsEmpty(t *testing.T) {
	// Empty menu
	menu := New([]string{})
	if !menu.IsEmpty() {
		t.Error("Expected empty menu to return true for IsEmpty()")
	}

	// Non-empty menu
	menu = New([]string{"Item 1"})
	if menu.IsEmpty() {
		t.Error("Expected non-empty menu to return false for IsEmpty()")
	}
}

func TestDefaultStyles(t *testing.T) {
	styles := DefaultStyles()

	if styles.Cursor != "  " {
		t.Errorf("Expected default cursor to be '  ', got '%s'", styles.Cursor)
	}
	if styles.SelectedCursor != "> " {
		t.Errorf("Expected default selected cursor to be '> ', got '%s'", styles.SelectedCursor)
	}

	// Test that styles are properly initialized (test by rendering)
	if styles.ItemStyle.Render("test") == "" {
		t.Error("Expected ItemStyle to be functional")
	}
	if styles.SelectedStyle.Render("test") == "" {
		t.Error("Expected SelectedStyle to be functional")
	}
}
