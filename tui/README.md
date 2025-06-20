# TUI Architecture

This document describes the clean architecture of the TUI (Terminal User Interface) package.

## Overview

The TUI package follows a clean architecture pattern with clear separation of concerns:

```
tui/
├── tui.go                    # Main TUI entry point (thin wrapper)
├── controller/               # Application controller (orchestration)
├── state/                    # State machine management
├── keys/                     # Key binding management
├── domain/                   # Business logic (project operations)
├── styles/                   # UI styling and theming
├── components/               # Reusable UI components
└── [feature-packages]/       # Feature-specific packages (login, projects, etc.)
```

## Architecture Principles

### 1. **Single Responsibility Principle**
Each package has a single, well-defined responsibility:
- `controller/` - Orchestrates the application flow
- `state/` - Manages state transitions
- `keys/` - Handles key bindings and input
- `domain/` - Contains business logic
- `styles/` - Manages UI appearance

### 2. **Dependency Injection**
All dependencies are injected through constructors, making the code testable and modular.

### 3. **State Machine Pattern**
The application uses a proper state machine with:
- Clear state definitions with documentation
- Validated state transitions
- State history tracking
- Centralized state management

### 4. **Command Pattern**
All side effects are handled through Bubble Tea commands, keeping the update logic pure.

## Package Details

### `tui.go` - Main Entry Point
A thin wrapper around the controller that implements the Bubble Tea interface.

```go
type Model struct {
    controller *controller.Controller
}
```

### `controller/` - Application Controller
The main orchestrator that:
- Manages application state
- Coordinates between components
- Handles state-specific logic
- Renders appropriate views

**Files:**
- `controller.go` - Main controller logic
- `commands.go` - Command functions (side effects)
- `views.go` - View rendering functions
- `version.go` - Version checking logic

### `state/` - State Machine
Provides a robust state machine with:
- Documented state definitions
- State transition validation
- History tracking
- Error handling

**States:**
- `RefreshingToken` - Refreshing user authentication
- `MainMenu` - Main application menu
- `Login` - User authentication screen
- `ProjectNameMenu` - Project name selection
- `ProjectVariantMenu` - Project variant selection
- `TestProject` - Project testing functionality

### `keys/` - Key Management
Centralizes key binding logic to eliminate duplication:
- Global key bindings (quit, navigation)
- Context-specific key handlers
- Footer binding helpers
- Consistent key behavior across the app

### `domain/` - Business Logic
Contains project-related business logic:
- `ProjectService` - API interactions
- `ProjectUtils` - Project manipulation utilities
- Domain-specific message types

### `styles/` - UI Styling
Centralized styling and theming:
- Color definitions
- Common styles
- ASCII art rendering
- Version information display

## Design Patterns Used

### 1. **State Machine Pattern**
- Clear state definitions with documentation
- Validated transitions
- History tracking
- Error handling

### 2. **Controller Pattern**
- Centralized application orchestration
- State-specific message handling
- View rendering coordination

### 3. **Strategy Pattern**
- Different key handlers for different contexts
- State-specific update strategies
- Context-appropriate footer bindings

### 4. **Dependency Injection**
- Constructor injection for all dependencies
- Interface-based dependencies
- Testable and modular design

### 5. **Command Pattern**
- All side effects as commands
- Pure update functions
- Composable command batching

## Benefits of This Architecture

### 1. **Maintainability**
- Clear separation of concerns
- Single responsibility principle
- Easy to locate and modify functionality

### 2. **Testability**
- Dependency injection enables easy mocking
- Pure functions are easy to test
- Clear interfaces for testing

### 3. **Extensibility**
- Easy to add new states
- Simple to add new components
- Clear patterns to follow

### 4. **Readability**
- Self-documenting code structure
- Clear naming conventions
- Comprehensive documentation

### 5. **Consistency**
- Centralized key handling eliminates duplication
- Consistent styling across the application
- Standardized patterns throughout

## Adding New Features

### Adding a New State
1. Add the state to `state/machine.go` with documentation
2. Add state-specific handler in `controller/controller.go`
3. Add view rendering in `controller/views.go`
4. Add appropriate key bindings in `keys/bindings.go`

### Adding a New Component
1. Create component in appropriate package
2. Inject dependencies through constructor
3. Implement standard Update/View interface
4. Add to controller if needed

### Adding Business Logic
1. Add to `domain/` package
2. Create service with injected dependencies
3. Define appropriate message types
4. Use from controller

This architecture ensures the codebase remains clean, maintainable, and follows Go best practices. 