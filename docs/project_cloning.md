# Project Cloning Feature

## Overview
The project cloning feature allows users to download project templates from GitHub based on their selected language preference. This feature is part of the project initialization workflow.

## Architecture

### States
- `stateProjectList`: Shows available projects with download status
- `stateLanguageSelection`: Shows available languages for the selected project
- Cloning progress is shown with a simulated progress bar

### Data Flow
1. User selects a project from the project list
   - Already downloaded projects are marked with a "✓ Downloaded" status
   - Attempting to select a downloaded project shows an error message
2. System parses available languages from the project's language field
3. User selects a language
4. System constructs GitHub repository URL using the format: `github.com/404skill/{project_name}_{language}`
5. Repository is cloned to `~/404skill_projects/{project_name}_{language}`
6. Project ID is added to the config file's `DownloadedProjects` map

### Error Handling
- Failed clone operations are displayed to the user
- User can retry the operation or go back to project selection
- Error messages are cleared when starting a new operation
- Attempting to download an already downloaded project shows an error message

## Implementation Details

### Project Status Display
- Projects are marked with a "✓ Downloaded" status in the project list
- Status is shown in a faint green color to indicate completion
- Downloaded projects cannot be re-initialized to prevent duplicates

### Repository URL Format
- Project names are converted to lowercase
- Spaces are replaced with underscores
- Language is appended with an underscore
- Example: "Scooter Rental System" with language "dotnet" becomes:
  `github.com/404skill/scooter_rental_system_dotnet`

### Progress Indication
- Since git clone doesn't provide real-time progress, we simulate progress updates
- Progress bar updates every 100ms
- Visual feedback helps users understand the operation is in progress

### Configuration Updates
- Downloaded projects are tracked in `~/.404skill/config.yml`
- Project IDs are stored in a map to prevent duplicate downloads
- Configuration is updated after successful clone operations
- Project status is checked before allowing initialization

## Future Improvements
1. Add support for custom repository URLs
2. Implement real progress tracking for git clone operations
3. Add option to update existing projects
4. Add support for different git providers
5. Implement parallel download capabilities for multiple projects
6. Add ability to remove projects from downloaded list
7. Add option to force re-download of existing projects 