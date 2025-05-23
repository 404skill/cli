# 404skill-cli System Overview

## Project Description
404skill-cli is a command-line interface tool for managing and interacting with the 404skill platform. It provides a terminal user interface (TUI) for various operations including project initialization, testing, and management.

## Core Components

### TUI (Terminal User Interface)
- Located in `tui/` directory
- Built using the Charm library (bubbletea)
- Provides interactive menus and project management interface
- Handles user authentication and session management

### API Client
- Located in `api/` directory
- Manages communication with the 404skill backend
- Handles project listing, creation, and management
- Implements proper error handling and response validation

### Authentication
- Located in `auth/` directory
- Integrates with Supabase for authentication
- Manages JWT tokens and session persistence
- Handles token refresh and expiration

### Configuration
- Located in `config/` directory
- Manages user preferences and credentials
- Stores configuration in `~/.404skill/config.yml`
- Handles environment-specific settings

## Architecture Principles
1. Modular design with clear separation of concerns
2. Stateless command execution
3. Proper error handling and user feedback
4. Secure credential management
5. Extensible command system

## Dependencies
- Charm libraries (bubbletea, lipgloss)
- Supabase client
- Standard Go libraries

## Configuration Requirements
- Supabase credentials
- Environment variables for different environments
- Local configuration file

## Security Considerations
- Secure credential storage
- JWT token management
- Input validation
- API call security

## Performance Considerations
- Minimal API calls
- Efficient TUI rendering
- Proper resource cleanup
- Timeout management 