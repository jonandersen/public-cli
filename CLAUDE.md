# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`pub` is a CLI for the Public.com trading API. It enables trading stocks, ETFs, options, and crypto from the command line using Public.com's personal access token authentication.

## Build Commands

```bash
make build              # Build the binary
make test               # Run tests with race detection and coverage
make lint               # Run golangci-lint
make fmt                # Format code with goimports and gofmt
make all                # Format, lint, test, and build
go run .                # Run without building
```

## Testing (TDD Approach)

**Write tests first.** This project follows TDD - always write failing tests before implementation.

```bash
go test ./...                    # Run all tests
go test -v -race -cover ./...    # With verbose, race detection, coverage
go test ./internal/config/...    # Test specific package
```

**Testing patterns:**
- Use `httptest.Server` for mocking HTTP responses
- Use `keyring.MockInit()` for mocking system keyring
- Use table-driven tests for comprehensive coverage
- Test Cobra commands with `cmd.SetOut()` and `cmd.SetArgs()`

## Architecture

```
pub/
├── cmd/                    # Cobra commands
│   ├── root.go            # Root command, config loading
│   ├── account.go         # Account commands
│   ├── quote.go           # Quote commands
│   ├── order.go           # Order commands
│   ├── options.go         # Options commands
│   └── configure.go       # First-time setup
├── internal/
│   ├── api/               # HTTP client with auth
│   ├── auth/              # Token exchange logic
│   ├── config/            # Config file management
│   ├── keyring/           # Secret storage abstraction
│   └── output/            # Table/JSON formatting
├── main.go                # Entry point
└── main_test.go           # Integration tests
```

## Key Dependencies

- `github.com/spf13/cobra` - CLI framework
- `github.com/zalando/go-keyring` - Cross-platform secure secret storage
- `github.com/stretchr/testify` - Test assertions and mocks
- `gopkg.in/yaml.v3` - Config file parsing

## Configuration

**Config file:** `~/.config/pub/config.yaml`
```yaml
account_uuid: "..."              # Default account
api_base_url: "https://api.public.com"
token_validity_minutes: 60
```

**Token cache:** `~/.config/pub/.token_cache` (chmod 600)
- Cached access tokens with TTL
- Auto-refresh when expired

**Secret storage:** System keyring (macOS Keychain, Linux Secret Service, Windows Credential Manager)
- Secret key stored securely, never in plain text

## Public.com API Notes

See `public-api-overview.md` for full API details.

**Authentication flow:**
1. User generates secret key at public.com/settings/security/api
2. `pub configure` stores secret key in system keyring
3. CLI exchanges secret key for access token (cached with TTL)
4. Access token used in `Authorization: Bearer` header

**Key endpoints:**
- `POST /userapiauthservice/personal/access-tokens` - Token exchange
- `GET /accounts` - List accounts
- `GET /accounts/{id}/portfolio` - Account portfolio
- `POST /quotes` - Get quotes
- `POST /orders` - Place orders

## Output Format

- Default: Human-readable tables
- `--json` flag: Machine-readable JSON for scripting

## Linting

Uses golangci-lint v2. Configuration in `.golangci.yml`.

```bash
make lint                        # Run linter
golangci-lint run --fix          # Auto-fix issues
```
