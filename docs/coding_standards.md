# Coding Standards

## General Principles

### 1. Code Organization
- Each file should have a single responsibility
- Related functionality should be grouped in the same package
- Keep files focused and manageable in size (ideally under 300 lines)
- Use clear and consistent file naming conventions

### 2. Naming Conventions
- Use camelCase for variables and functions
- Use PascalCase for types and interfaces
- Use UPPER_CASE for constants
- Use descriptive names that indicate purpose
- Avoid abbreviations unless widely understood

### 3. Documentation
- Every exported type, function, and method must have documentation
- Documentation should explain the purpose and behavior
- Include examples for complex functionality
- Document any assumptions or limitations
- Keep documentation up to date with code changes

### 4. Error Handling
- Always check and handle errors
- Provide meaningful error messages
- Use custom error types when appropriate
- Log errors for debugging purposes
- Handle errors at the appropriate level

## Component Guidelines

### 1. Component Structure
```go
// ComponentName handles specific functionality
type ComponentName struct {
    // State fields
    state    State
    errorMsg string
    
    // Dependencies
    client   ClientInterface
    manager  ManagerInterface
}

// NewComponentName creates a new component
func NewComponentName(client ClientInterface, manager ManagerInterface) *ComponentName {
    return &ComponentName{
        client:  client,
        manager: manager,
    }
}

// Update handles messages for the component
func (c *ComponentName) Update(msg tea.Msg) (Component, tea.Cmd) {
    // Implementation
}

// View renders the component
func (c *ComponentName) View() string {
    // Implementation
}
```

### 2. State Management
- Keep state immutable
- Use messages for state transitions
- Document state changes
- Handle all possible states
- Provide clear feedback for state changes

### 3. Error Handling
```go
// Example error handling pattern
if err != nil {
    return errMsg{err: fmt.Errorf("failed to perform operation: %w", err)}
}
```

## Interface Design

### 1. Interface Definition
```go
// InterfaceName defines the contract for a specific functionality
type InterfaceName interface {
    // Method1 performs a specific operation
    Method1() error
    
    // Method2 performs another operation
    Method2() (Result, error)
}
```

### 2. Interface Implementation
```go
// ImplementationName implements InterfaceName
type ImplementationName struct {
    // Dependencies
    client ClientInterface
}

// Method1 implements InterfaceName.Method1
func (i *ImplementationName) Method1() error {
    // Implementation
}
```

## Testing Standards

### 1. Unit Tests
- Test each component in isolation
- Mock dependencies
- Test all edge cases
- Use table-driven tests
- Keep tests focused and readable

### 2. Test Structure
```go
func TestComponentName_Operation(t *testing.T) {
    tests := []struct {
        name     string
        input    Input
        expected Output
        wantErr  bool
    }{
        // Test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

## UI/UX Guidelines

### 1. User Feedback
- Provide clear feedback for all actions
- Show progress for long-running operations
- Display meaningful error messages
- Use consistent styling
- Handle edge cases gracefully

### 2. Styling
- Use the centralized style definitions
- Maintain consistent color schemes
- Follow the established layout patterns
- Ensure readability
- Support different terminal sizes

## Git Workflow

### 1. Branching
- Use feature branches for new features
- Use bugfix branches for bug fixes
- Keep branches focused and short-lived
- Use descriptive branch names

### 2. Commits
- Write clear commit messages
- Keep commits focused and atomic
- Reference issues in commit messages
- Follow conventional commit format

### 3. Pull Requests
- Include clear description
- Reference related issues
- Include testing instructions
- Request reviews from team members
- Address review comments

## Code Review Guidelines

### 1. What to Review
- Code correctness
- Error handling
- Documentation
- Testing coverage
- Performance implications
- Security considerations

### 2. Review Process
- Be constructive and respectful
- Focus on the code, not the person
- Provide clear explanations
- Suggest improvements
- Verify changes address the issue

## Performance Guidelines

### 1. Optimization
- Profile before optimizing
- Focus on bottlenecks
- Consider memory usage
- Optimize for common cases
- Document performance characteristics

### 2. Resource Management
- Close resources properly
- Handle cleanup in defer statements
- Use appropriate buffer sizes
- Consider concurrent operations
- Monitor resource usage

## Security Guidelines

### 1. Data Handling
- Validate all input
- Sanitize output
- Handle sensitive data securely
- Use secure defaults
- Follow principle of least privilege

### 2. Authentication
- Use secure authentication methods
- Handle tokens securely
- Implement proper session management
- Log security events
- Follow security best practices 