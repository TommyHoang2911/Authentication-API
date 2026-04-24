# Auth Service API Documentation

## Overview
This service implements authentication and QR login flows using Gin.

- Base URL: `/`
- Authentication: JWT in `Authorization: Bearer <token>` for protected routes
- Content-Type: application/json
- **Email Confirmation Required**: Users must confirm their email before login

---

## API Reference

| Function | Endpoint | Request | Response | Note |
| --- | --- | --- | --- | --- |
| Register user | `POST /register` | `email` (string, required, email format)<br>`password` (string, required, min 8 chars) | `201`<br>`{ "message": "user registered successfully", "user": { "id": 1, "email": "..." } }` | Confirmation email sent automatically; user must confirm email before login |
| Confirm email | `POST /confirm_email` | `token` (string, required) | `200`<br>`{ "message": "email confirmed successfully" }` | Validates email confirmation token |
| Resend confirmation | `POST /resend_confirmation` | `email` (string, required, email format) | `200`<br>`{ "message": "confirmation email sent successfully" }` | Fails if email not found or already confirmed |
| Login | `POST /login` | `email` (string, required)<br>`password` (string, required) | `200`<br>`{ "message":"login successful", "user": {"id":1,"email":"..."}, "token":"<jwt>", "refresh_token":"<token>" }` | Requires confirmed email; returns access and refresh tokens |
| Generate QR | `POST /generate_qr` | `device_id` (string, required) | `200`<br>`{ "code": "<qr_code_hash>" }` | Creates a QR auth code for device-based login |
| Exchange code | `POST /exchange_code` | `temp_code` (string, required) | `200`<br>`{ "message":"code exchanged successfully", "user": {...}, "token":"<jwt>", "session_token":"<refresh>" }` | Completes Device B login after QR verification |
| WebSocket status | `GET /ws` | N/A | WebSocket connection for QR status updates | No JSON response |
| Get current user | `GET /user` | Bearer token required | `200`<br>`{ "user": {"id":1, "email":"..."} }` | Protected route requiring JWT |
| Sign out | `POST /sign_out` | `refresh_token` (string, required) | `200`<br>`{ "message":"signed out successfully" }` | Invalidates refresh token |
| Refresh token | `POST /refresh_token` | `refresh_token` (string, required) | `200`<br>`{ "token":"<new_jwt>" }` | Issues a new JWT from a valid refresh token |
| Verify QR | `POST /verify_qr` | `code` (string, required) | `200`<br>`{ "message":"QR code verified successfully" }` | Protected route; sends temp code via WebSocket |

---

## Notes
- QR flow: `generate_qr` (Device B) -> `verify_qr` (Device A, protected) -> `exchange_code` (Device B)
- All endpoints return errors in `{ "error": "..." }` on failure.
