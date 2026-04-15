# Auth Service API Documentation

## Overview
This service implements authentication and QR login flows using Gin.

- Base URL: `/`
- Authentication: JWT in `Authorization: Bearer <token>` for protected routes
- Content-Type: application/json
- **Email Confirmation Required**: Users must confirm their email before login

---

## Public Endpoints

### `POST /register`
Register a new user.

Request:
- `email` (string, required, email format)
- `password` (string, required, min 8 chars)

Response 201:
```json
{ "message": "user registered successfully", "user": { "id": 1, "email": "..." }}
```

**Note:** User must confirm email before login. Confirmation email sent automatically.

Error 400 on invalid input or existing user.

### `POST /confirm_email`
Confirm user email using token from confirmation email.

Request:
- `token` (string, required)

Response 200:
```json
{ "message": "email confirmed successfully" }
```

Error 400 on invalid/expired token.

### `POST /resend_confirmation`
Resend confirmation email to user.

Request:
- `email` (string, required, email format)

Response 200:
```json
{ "message": "confirmation email sent successfully" }
```

Error 400 if email not found or already confirmed.

### `POST /login`
Authenticate user and issue tokens.

**Note:** User must have confirmed email to login.

Request:
- `email` (string, required)
- `password` (string, required)

Response 200:
```json
{
  "message":"login successful",
  "user": {"id":1,"email":"..."},
  "token":"<jwt>",
  "refresh_token":"<token>"
}
```

Error 401 on invalid credentials or unconfirmed email.

### `POST /generate_qr`
Create a device QR auth code.

Request:
- `device_id` (string, required)

Response 200:
```json
{ "code": "<qr_code_hash>" }
```

### `POST /exchange_code`
Exchange a temporary code (from Device B) for tokens.

Request:
- `temp_code` (string, required)

Response 200:
```json
{
  "message":"code exchanged successfully",
  "user": {...},
  "token":"<jwt>",
  "session_token":"<refresh>"
}
```

### `GET /ws`
WebSocket endpoint (no JSON) for QR status updates.

---

## Protected Endpoints (Bearer JWT required)

- `Authorization: Bearer <token>`
- `JWTAuthMiddleware()` is applied.

### `GET /user`
Get current user profile.

Response 200:
```json
{ "user": {"id":1, "email":"..."} }
```

### `POST /sign_out`
Invalidate refresh token (logout).

Request:
- `refresh_token`: string, required

Response 200:
```json
{ "message":"signed out successfully" }
```

### `POST /refresh_token`
Refresh JWT using refresh token.

Request:
- `refresh_token`: string, required

Response 200:
```json
{ "token":"<new_jwt>" }
```

### `POST /verify_qr`
Verify a generated QR code and send temp code via websocket.

Request:
- `code`: string, required

Response 200:
```json
{ "message":"QR code verified successfully" }
```

---

## Notes
- QR flow: `generate_qr` (Device B) -> `verify_qr` (device A, protected) -> `exchange_code` (Device B)
- All endpoints return errors in `{ "error": "..."}` on failure.
