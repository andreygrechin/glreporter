# glreporter

glreporter is the CLI tool to fetch information from GitLab groups and projects.

[![license](https://img.shields.io/badge/License-MIT-blue.svg)](https://github.com/andreygrechin/glreporter/blob/main/LICENSE)

## Features

- Collect information about GitLab groups and their projects.
- Filter by group ID and project status.
- Output in a JSON, table, or CSV format.

## Installation

### go install

```shell
go install github.com/andreygrechin/glreporter@latest
```

### Manually

Download the pre-compiled binaries from [the releases page](https://github.com/andreygrechin/glreporter/releases/) and copy them to a desired location.

## Usage

### Available Commands

```shell
# View help for all commands
glreporter --help

# Generate shell completion
glreporter completion <shell>
```

### Groups and Projects

```shell
# Fetch groups recursively
glreporter groups --group-id <group-id>

# Fetch projects from all groups
glreporter projects --group-id <group-id>
```

### Token Management

```shell
# Fetch group access tokens (from all subgroups by default)
glreporter tokens gat --group-id <group-id>

# Include inactive group access tokens
glreporter tokens gat --group-id <group-id> --include-inactive

# Fetch project access tokens for all projects in a group
glreporter tokens pat --group-id <group-id>

# Fetch project access tokens for a specific project
glreporter tokens pat --project-id <project-id>

# Include inactive project access tokens
glreporter tokens pat --group-id <group-id> --include-inactive

# Fetch pipeline trigger tokens for all projects in a group
glreporter tokens ptt --group-id <group-id>

# Fetch pipeline trigger tokens for a specific project
glreporter tokens ptt --project-id <project-id>
```

### Global Flags

```shell
--format <format>     # Output format: table (default), json, or csv
--token <token>       # GitLab personal access token (or use GITLAB_TOKEN env var)
--debug               # Enable debug logging
```

### Command-Specific Flags

```shell
--group-id <group-id>   # GitLab group ID (required for groups/projects commands)
--project-id <project-id> # GitLab project ID (alternative to group-id for token commands)
--include-inactive      # Include inactive tokens in output (token commands only)
```

**Note**: For token commands, use either `--group-id` (to fetch from all projects in a group) or `--project-id` (to fetch from a specific project), but not both.

### Output Formats

- **Table**: Human-readable format with limited fields
- **JSON/CSV**: Complete raw API response data

### Authentication

The tool authenticates with the GitLab API using a personal access token (PAT). The token can be provided via:

- Command-line flag: `--token <token>`
- Environment variable: `GITLAB_TOKEN`

## License

This project is licensed under the [MIT License](LICENSE).

`SPDX-License-Identifier: MIT`
