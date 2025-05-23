# Project Dependencies

## Core Dependencies

### Charm Libraries
- **bubbletea**: Terminal UI framework
  - Version: Latest stable
  - Purpose: TUI implementation
  - Key features: State management, component system
  - Documentation: https://github.com/charmbracelet/bubbletea

- **lipgloss**: Terminal styling
  - Version: Latest stable
  - Purpose: UI styling and formatting
  - Key features: Colors, borders, layouts
  - Documentation: https://github.com/charmbracelet/lipgloss

- **bubbles**: UI components
  - Version: Latest stable
  - Purpose: Reusable TUI components
  - Key features: Text input, help, viewport
  - Documentation: https://github.com/charmbracelet/bubbles

### Supabase
- **supabase-go**: Supabase client
  - Version: Latest stable
  - Purpose: Authentication and backend integration
  - Key features: JWT handling, API client
  - Documentation: https://github.com/supabase-community/supabase-go

## Development Dependencies

### Testing
- **testify**: Testing framework
  - Version: Latest stable
  - Purpose: Unit testing
  - Key features: Assertions, mocking
  - Documentation: https://github.com/stretchr/testify

### Code Quality
- **golangci-lint**: Linter
  - Version: Latest stable
  - Purpose: Code quality checks
  - Key features: Multiple linters, custom rules
  - Documentation: https://github.com/golangci/golangci-lint

## Version Management
- Go modules for dependency management
- Version pinning for stability
- Regular dependency updates
- Security vulnerability scanning

## Dependency Update Policy
1. Regular security updates
2. Major version updates require testing
3. Breaking changes must be documented
4. Dependencies must be justified

## Security Considerations
1. Regular security audits
2. Dependency vulnerability scanning
3. Minimal dependency footprint
4. Trusted sources only

## Build Requirements
1. Go 1.21 or later
2. Git for version control
3. Make for build automation
4. golangci-lint for code quality

## Runtime Requirements
1. Terminal with UTF-8 support
2. Minimum terminal size: 80x24
3. Network access for API calls
4. Local file system access 