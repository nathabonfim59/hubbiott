# AGENTS.md

Guidelines for AI coding agents working on Hubbiott.

## Commands

```bash
go build -o bin/server ./cmd/server    # Build
go run ./cmd/server                     # Run server
go test ./...                           # All tests
go test -run TestName ./path            # Specific test
go test -v -cover ./...                 # Verbose + coverage
golangci-lint run                       # Lint
go fmt ./... && go vet ./...            # Format + vet
go mod tidy                             # Tidy deps

cd frontend && npm run dev/build/test/lint
```

## Structure

```
cmd/server/main.go        # Entry point
internal/
├── api/                  # HTTP handlers
├── bot/                  # Discord bot
├── config/               # Config
├── db/                   # Database
├── queue/                # Job queue
└── services/             # Business logic
frontend/                 # SvelteKit
migrations/               # SQL migrations
```

## Go Style

**Imports**: stdlib → third-party → local (blank lines between)

**Naming**: Packages lowercase, files snake_case, types/exports PascalCase, unexported camelCase, errors `Err` prefix, interfaces `-er` suffix

**Errors**: Always handle explicitly, wrap with `fmt.Errorf`, return early

**Structs**: Use named fields always

**Context**: First param for I/O functions, never store in structs

**Interfaces**: Keep small and focused

## Testing

- Files: `*_test.go`, functions: `Test<Name>_<Scenario>`
- Table-driven tests with `t.Run()`
- Arrange-Act-Assert pattern

## Frontend

Components PascalCase, routes lowercase-dashed, use TypeScript

## Architecture

- **Job Queue**: Webhooks <10s, long ops (AI) queued, sequential workers
- **OpenCode Manager**: TTL lifecycle, auto-start, idle shutdown, call `EnsureRunning()` first
- **Database**: Turso/libSQL + SQLite fallback, embedded migrations, use transactions
- **Auth**: HttpOnly/Secure/SameSite=Strict cookies, middleware validation

## Dependencies

`labstack/echo` (HTTP), `bwmarrin/discordgo` (Discord), `tursodatabase/libsql-client-go` (DB), `google/go-github` (GitHub), `x/crypto/bcrypt`

## Pre-Commit

1. `go fmt ./...`
2. `go vet ./...`
3. `go test ./...`
4. `golangci-lint run`
5. `npm run lint` and `npm run check`  (frontend/)
