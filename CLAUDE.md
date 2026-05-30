# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

All commands must be run from the `auth-service/` directory.

```sh
# Run the server
go run ./cmd/server

# Run all tests
go test ./...

# Run tests with coverage
go test ./... -cover

# Run a single package's tests
go test ./internal/service/...
go test ./internal/handler/...

# Run a single test by name
go test ./internal/handler/... -run TestAuthHandler_Login

# Static analysis
go vet ./...

# Regenerate Swagger docs (requires swag CLI)
swag init -g cmd/server/main.go
```

Docker Compose (from `auth-service/`):

```sh
docker compose up --build     # start with rebuild
docker compose up --build -d  # detached
docker compose down
docker compose logs -f auth_service
```

The Docker Compose stack reads config from `.env.docker`. The app connects to a Postgres container (`auth_postgres`) that must be healthy before the API starts.

## Architecture

The codebase lives entirely under `auth-service/` and follows a layered architecture:

```
cmd/server/main.go          → wires everything together, runs HTTP + graceful shutdown
internal/config/            → env/config loading + DB init + migration runner
internal/domain/            → plain structs (User, AuthCode) — no logic
internal/handler/           → HTTP handlers (Gin), request/response types, error mapping
internal/middleware/        → JWT auth middleware (injects user_id into Gin context)
internal/repository/        → SQL queries against PostgreSQL (UserStore, AuthCodeStore interfaces)
internal/service/           → business logic, split into sub-services:
  auth_service.go           → facade that delegates to UserService, TokenService, QRService, OAuthService
  user_service.go           → registration, login, email confirmation
  token_service.go          → JWT + refresh token lifecycle
  qr_service.go             → QR code generation, verification, temp-code exchange
  oauth_service.go          → Google/Facebook OAuth2 flows
  email_service.go          → SMTP email sender
  queue/email_queue.go      → async email worker pool with linear-backoff retry
internal/transport/http/    → Gin router setup (all routes declared here)
internal/transport/ws/      → WebSocket hub (maps QR codes to open connections)
internal/testutil/          → shared test helper (sqlmock factory)
pkg/jwt/                    → JWT Manager interface + HMAC implementation
db/migrations/              → numbered up/down SQL migrations (auto-applied on startup)
```

### Dependency flow

```
Handler → AuthServiceInterface (defined in handler/types.go)
       ↓
AuthService (facade) → UserService → UserStore (repository interface)
                     → TokenService
                     → QRService → AuthCodeStore + WebSocket Hub
                     → OAuthService
```

The `AuthServiceInterface` in `internal/handler/types.go` is the boundary between handlers and services. Handlers never import concrete service types — only this interface, which makes handler tests easy to mock.

### QR login flow (cross-device)

1. Device B calls `POST /generate_qr` with a `device_id` → gets a code and opens `GET /ws?code=<code>` WebSocket
2. Device A (authenticated) calls `POST /verify_qr` with the code → server marks the code used, creates a short-lived `temp_code`, and pushes it to Device B over WebSocket
3. Device B calls `POST /exchange_code` with the `temp_code` → gets full JWT tokens

The WebSocket Hub (`internal/transport/ws/hub.go`) maps QR codes to open connections using a `sync.RWMutex`-protected map.

### Error handling

All errors are returned as `AppError` from `internal/errors/errors.go`. Each error has a typed code with a category prefix:

- `AUTH_*`, `USC_*` (user service), `TKN_*` (token), `QR_*`, `OAUTH_*`, `DB_*`, `INT_*`, `SYS_*`, `SOCK_*`, `DLV_*` (validation), `CFG_*`

Handlers call `RespondWithError(c, statusCode, err)` which serializes as `{"err_code": "...", "err_message": "..."}`. Non-`AppError` values get `err_code: "SYS_ERROR"`.

### Configuration

Config loads from `.env` (via godotenv) and environment variables, always validated on startup. Swagger is always enabled and requires:

- `SWAGGER_HOST_{APP_ENV}` (e.g. `SWAGGER_HOST_DEVELOPMENT=localhost:8080`)
- `SWAGGER_USERNAME` and `SWAGGER_PASSWORD` (basic auth)

`JWT_SECRET` is required and must not equal `"your-secret-key"`. `DATABASE_URL` takes precedence over individual `DB_*` vars.

### Testing approach

- **Handler tests**: Use `MockAuthService` (testify/mock, in `handler/mock_service_test.go`). Each test creates a `gin.TestContext` with `httptest.NewRecorder`.
- **Service/repository tests**: Use `go-sqlmock` via `testutil.NewSQLMock(t)` which registers cleanup automatically.
- **Handler tests run without a database** — the mock implements `AuthServiceInterface`.

## CI/CD

- **PRs to master**: runs `go test -v -cover ./...` and `go vet ./...`, then builds the Docker image (push disabled).
- **Push to master**: runs tests, then builds and pushes to GCP Artifact Registry, deploys to Cloud Run (`asia-east2`). Secrets (JWT_SECRET, DATABASE_URL, SMTP_PASS, etc.) come from GCP Secret Manager.

## Swagger

Swagger docs live in `docs/` (generated by `swag`). The entrypoint annotations are in `cmd/server/main.go`. After changing handler annotations, regenerate with `swag init -g cmd/server/main.go` from `auth-service/`.

Swagger UI is at `/swagger/index.html` and requires HTTP basic auth in all environments.
