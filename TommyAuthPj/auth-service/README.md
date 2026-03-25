# Auth Service

This service provides user registration and authentication backed by PostgreSQL.

## Features

- User registration and login
- JWT-based authentication
- QR code device binding
- **Email confirmation required for login**

## Setup

1. Install PostgreSQL and create a database, e.g.: 

```sh
createdb authdb
```

2. Run the migration script:

```sh
psql -d authdb -f config/migrations.sql
```

3. Set `DATABASE_URL` environment variable (optional). Default is:
   `postgres://postgres:password@localhost:5432/authdb?sslmode=disable`.

4. Configure SMTP settings for email confirmation (see .env.example)

5. Build and run the service:

```sh
go build ./...
./auth-service/server/main
```

The server listens on port `8080` by default.

## Email Confirmation

Users must confirm their email address before they can log in. After registration, an email with a confirmation link is sent. Users need to click the link to activate their account.

### API Endpoints

- `POST /register` - Register a new user (sends confirmation email)
- `POST /confirm-email` - Confirm email with token from email link
- `POST /resend-confirmation` - Resend confirmation email
- `POST /login` - Login (only works for confirmed users)

## Generating QR Code for Device Binding

Send a POST request to `/generate_qr` with JSON body containing the device_id:

```json
{
  "device_id": "device-123"
}
```

This will generate a hashed QR code that incorporates the device_id for security. The QR code can be scanned by an authenticated user to bind the device to their account.

## Verifying QR Code

Authenticated users can send a POST request to `/verify_qr` with the scanned code:

```json
{
  "code": "scanned-qr-code"
}
```

This will verify the code and send a temporary code back to the device via WebSocket for token exchange.

## Notes

- Passwords are stored in plaintext for now. Replace with hashing in production.
- Add more fields or validation as needed.
