# 404skill CLI

## Overview

The 404skill CLI is a command-line interface tool designed to interact with the 404skill backend. It provides functionalities for user authentication, project listing, and more.

## Key Features

- **User Authentication**: Handles user login and token management via supabase.
- **Project Listing**: Fetches and displays project metadata in a table format.

## Implementation Details

### Authentication

- **Token Management**: The CLI uses JWT tokens for authentication. Tokens are stored in a configuration file and refreshed automatically if expired. Apart from the token, we also store the username, password, and last_update time of the token, so that we can automatically detect if the token needs to be refreshed, so the user doesn't have to login again (we login for him).

### Configuration

- **Environment Variables**: The CLI uses environment variables to manage different configurations for development and production environments.
- **Base URL**: The base URL for API calls is determined by the `ENV` environment variable, which can be set to `production` or any other value for development.


## Usage

Help Options:  
  /?          Show this help message  
  /h, /help   Show this help message  

Available commands:  
  init   Initialize a project  
  list   List projects  
  login  Login to 404skill  
  test   Run tests  