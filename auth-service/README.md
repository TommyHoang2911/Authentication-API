# Auth Service

Auth Service is a Go-based authentication backend with JWT auth, OAuth login, email confirmation, and QR/WebSocket login flow.

## Features

- Email/password registration and login
- JWT access token + refresh token flow
- Email confirmation and resend confirmation
- OAuth login (Google and Facebook)
- QR login flow with WebSocket handoff
- Swagger/OpenAPI documentation
- Async email sending queue with retry strategy
- PostgreSQL persistence with migrations

## Project Structure

- `cmd/server` - application entrypoint
- `internal/config` - environment and runtime config
- `internal/domain` - domain models
- `internal/handler` - HTTP handlers
- `internal/middleware` - auth middleware
- `internal/repository` - database repositories
- `internal/service` - business logic
- `internal/service/queue` - async email worker queue
- `internal/transport/http` - router setup
- `internal/transport/ws` - WebSocket hub
- `pkg/jwt` - JWT manager abstraction
- `db/migrations` - SQL migrations

## Requirements

- Go 1.25+
- PostgreSQL 14+

## Configuration

The service reads environment variables from your shell and `.env` (via `godotenv`).

Core variables:

- `APP_ENV` (`development`, `testing`, `production`)
- `SERVER_PORT` (default `8080`)
- `JWT_SECRET` (required)
- `BASE_URL` (used for email confirmation links)
- `DATABASE_URL` (recommended)

If `DATABASE_URL` is not set, the service builds one from:

- `DB_USER`, `DB_PASSWORD`, `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_SSLMODE`

SMTP variables:

- `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASS`, `SMTP_FROM`

OAuth variables:

- `GOOGLE_OAUTH_CLIENT_ID`, `GOOGLE_OAUTH_CLIENT_SECRET`, `GOOGLE_OAUTH_REDIRECT_URL`
- `FACEBOOK_OAUTH_CLIENT_ID`, `FACEBOOK_OAUTH_CLIENT_SECRET`, `FACEBOOK_OAUTH_REDIRECT_URL`

Swagger host (required when swagger is enabled):

- `SWAGGER_HOST_DEVELOPMENT` or `SWAGGER_HOST_TESTING` based on `APP_ENV`

Email queue variables:

- `EMAIL_QUEUE_WORKERS` (default `2`)
- `EMAIL_QUEUE_BUFFER_SIZE` (default `100`)
- `EMAIL_QUEUE_MAX_RETRIES` (default `3`)
- `EMAIL_QUEUE_RETRY_DELAY_SECONDS` (default `1`)

## Run Locally

1. Create PostgreSQL database.
2. Configure `.env` values.
3. Start the service:

```sh
go run ./cmd/server
```

The service runs on `http://localhost:8080` by default.

Migrations are applied automatically on startup from `db/migrations`.

4. Configure SMTP settings for email confirmation, including `SMTP_TIMEOUT_SECONDS` for SMTP network timeouts (default `10`).

## Docker

If you use Docker Compose in this repository:

```sh
docker compose up --build
```

## API Documentation

Swagger UI is enabled when `APP_ENV != production`.

- Swagger route: `/swagger/index.html`

## Authentication Flows

Email/password flow:

1. `POST /register`
2. Confirm email from confirmation link
3. `POST /login`
4. `POST /refresh-token`
5. `POST /signout`

OAuth flow:

1. `GET /auth/{provider}`
2. `GET /auth/{provider}/callback`

QR login flow:

1. Device B calls `POST /generate_qr`
2. Device A (authenticated) calls `POST /verify_qr`
3. Device B exchanges temporary code via `POST /exchange_code`

## Email Queue Behavior

- Registration and resend email requests enqueue jobs (non-blocking SMTP path).
- Workers process jobs asynchronously.
- Failed jobs retry with linear backoff.
- Jobs that exceed max retries are logged and dropped.
- Queue performs graceful shutdown and drains in-flight work.

## Testing

Run all tests:

```sh
go test ./...
```

Run with coverage:

```sh
go test ./... -cover
```
