# Privilege Escalation Management System

A system for managing privilege escalation and temporary access control in enterprise environments.

## Overview

This system provides a secure and auditable way to manage privilege escalation and temporary access requests. It helps organizations maintain security while providing necessary access to users who need elevated privileges for specific tasks.

## Architecture

The system is split into three main components:

### 1. CLI (Command Line Interface)
- Handles user authentication via OpenID Connect (Google)
- Provides command-line interface for privilege management
- Communicates with the API server
- Manages local credentials and tokens

### 2. API Server
- Central management system for privilege escalations
- Handles operator registration and management
- Manages privilege request workflow
- Emits events for system state changes
- Integrates with notification systems (e.g., Slack)

### 3. Operators
- Modular components deployed near target systems
- Handles specific privilege escalation tasks
- Examples:
  - MySQL temporary user management
  - Kubernetes role management
  - AWS IAM role management

## Project Structure

```
apollo/
├── cmd/
│   ├── cli/            # CLI application
│   ├── api/            # API server
│   └── operator/       # Operator base implementation
├── internal/
│   ├── core/           # Core business logic
│   ├── auth/           # Authentication logic
│   ├── events/         # Event system
│   ├── operators/      # Operator implementations
│   └── pkg/            # Internal packages
├── pkg/                # Public packages
├── configs/            # Configuration files
├── test/               # Test files
└── docs/               # Documentation
```

## Prerequisites

- Go 1.21 or later
- PostgreSQL 13 or later
- Redis 6 or later
- Google Cloud Platform account (for OpenID Connect)

## Setup

1. Clone the repository:
   ```bash
   git clone [repository-url]
   cd apollo
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Configure the system:
   ```bash
   cp configs/config.example.yaml configs/config.yaml
   # Edit config.yaml with your settings
   ```

4. Set up Google OAuth credentials:
   ```bash
   # Follow Google Cloud Console instructions to create OAuth 2.0 credentials
   # Add credentials to config.yaml
   ```

5. Run the components:

   API Server:
   ```bash
   go run cmd/api/main.go
   ```

   CLI:
   ```bash
   go run cmd/cli/main.go
   ```

   Operator:
   ```bash
   go run cmd/operator/main.go
   ```

## Development

- Run tests:
  ```bash
  go test ./...
  ```

- Run linter:
  ```bash
  golangci-lint run
  ```

## Security Considerations

- All privilege escalations are logged and auditable
- Temporary access is automatically revoked after the specified duration
- Access requests require approval from authorized personnel
- All actions are encrypted and secure
- OpenID Connect for secure authentication
- Event-based architecture for audit trail

## License

[License details to be added] 