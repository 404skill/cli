# Project Cloning Feature

## Overview
The project cloning feature allows users to download and manage programming projects from the 404skill platform. It provides a seamless experience for selecting, downloading, and accessing projects, with built-in support for multiple programming languages and error handling.

## Architecture

### States
The feature uses a state machine with the following states:
- `stateProjectList`: Displays available projects in a table format
- `stateLanguageSelection`: Allows users to select a programming language for the project
- `stateConfirmRedownload`: Handles re-downloading of projects when the directory is not found

### Data Model
The project table displays the following information:
- Project Name
- Available Languages
- Difficulty Level
- Estimated Duration
- Download Status (✓ Downloaded)

## Data Flow

### Project Selection and Download
1. User selects a project from the table
2. System checks if the project is already downloaded:
   - If downloaded: Attempts to open the project directory
   - If directory not found: Prompts for re-download
   - If not downloaded: Proceeds to language selection
3. User selects a programming language
4. System clones the repository with progress tracking
5. On successful clone:
   - Updates project status in the table
   - Opens the project directory in file explorer
   - Updates configuration to track downloaded projects

### Re-download Process
1. When a project is marked as downloaded but the directory is not found:
   - System enters `stateConfirmRedownload`
   - User can select a language for re-download
   - Existing directory is removed before cloning
   - Progress is tracked during the clone operation

## Error Handling

### Directory Management
- Projects are stored in `~/404skill_projects`
- Directory names are formatted as `project_name_language`
- Existing directories are removed before re-download
- File explorer opens automatically after successful clone

### Error Scenarios
1. Home Directory Access:
   - Error: "Project already downloaded but couldn't determine home directory"
   - Action: Prompts for re-download

2. Project Directory Not Found:
   - Error: "Project was downloaded but directory not found"
   - Action: Prompts for re-download with language selection

3. Directory Access Issues:
   - Error: "Project already downloaded but couldn't access projects directory"
   - Action: Shows error message

4. File Explorer Issues:
   - Error: "Project was downloaded but couldn't open directory"
   - Action: Shows error message but maintains download status

### Clone Operation
- Progress tracking using git clone's `--progress` flag
- Real-time progress updates from stderr output
- Error capture from git output
- Verification of successful clone by checking directory existence

## Implementation Details

### Progress Tracking
- Uses git clone's `--progress` flag for real-time updates
- Parses progress from stderr output
- Updates progress bar based on actual download progress
- Handles both initial downloads and re-downloads

### State Management
- Tracks both `selectedProject` and `confirmRedownloadProject`
- Clears project references after successful clone
- Updates table status without full refresh
- Maintains language selection state during re-download

### Configuration
- Tracks downloaded projects in config file
- Updates project status immediately after successful clone
- Persists download status across sessions

## Future Improvements
1. Add support for:
   - Project updates/pulling latest changes
   - Custom download locations
   - Project deletion
   - Multiple language downloads
2. Enhance error handling:
   - Network timeout handling
   - Retry mechanisms
   - More detailed error messages
3. Improve progress tracking:
   - More granular progress updates
   - Estimated time remaining
   - Download speed information 