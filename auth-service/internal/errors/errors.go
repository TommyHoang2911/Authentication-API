package apperrors

import (
	"fmt"
	"strings"
)

const (
	API_ERRORS                 = "API"   // API errors category
	AUTH_SERVICE_ERRORS        = "AUTH"  // Authentication Service errors category
	USER_SERVICE_ERRORS        = "USC"   // User Service errors category
	TOKEN_SERVICE_ERRORS       = "TKN"   // Token Service errors category
	QR_SERVICE_ERRORS          = "QR"    // QR Service errors category
	DELIVERY_VALIDATION_ERRORS = "DLV"   // Delivery/Validation errors category
	GRPC_ERRORS                = "GRPC"  // gRPC errors category
	DATABASE_ERRORS            = "DB"    // Database errors category
	INTERNAL_ERRORS            = "INT"   // Internal errors category
	SYSTEM_ERRORS              = "SYS"   // System errors category
	SOCKET_ERRORS              = "SOCK"  // Socket/WebSocket errors category
	CONFIGURATION_ERRORS       = "CFG"   // Configuration errors category
	OAUTH_SERVICE_ERRORS       = "OAUTH" // OAuth Service errors category
)

// AppError captures a typed application error with an optional wrapped cause.
type AppError struct {
	Code    string
	Message string
	Cause   error
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	if e.Cause == nil {
		return e.Message
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func New(category, code, message string, cause error) *AppError {
	return &AppError{Code: category + "_" + strings.ToUpper(code), Message: message, Cause: cause}
}

func NewValidation(message string) *AppError {
	return New(DELIVERY_VALIDATION_ERRORS, "VALIDATION_ERROR", message, nil)
}

func NewConfiguration(action, message string, cause error) *AppError {
	return New(CONFIGURATION_ERRORS, action+"_ERROR", message, cause)
}

func NewInternal(action, message string, cause error) *AppError {
	return New(INTERNAL_ERRORS, action+"_ERROR", message, cause)
}

func NewDatabase(action, message string, cause error) *AppError {
	return New(DATABASE_ERRORS, action+"_ERROR", message, cause)
}

func NewGRPC(action, message string, cause error) *AppError {
	return New(GRPC_ERRORS, action+"_ERROR", message, cause)
}

func NewAPI(action, message string, cause error) *AppError {
	return New(API_ERRORS, action+"_ERROR", message, cause)
}

func NewAuthService(action, message string, cause error) *AppError {
	return New(AUTH_SERVICE_ERRORS, action+"_ERROR", message, cause)
}

func NewUserService(action, message string, cause error) *AppError {
	return New(USER_SERVICE_ERRORS, action+"_ERROR", message, cause)
}

func NewTokenService(action, message string, cause error) *AppError {
	return New(TOKEN_SERVICE_ERRORS, action+"_ERROR", message, cause)
}

func NewQRService(action, message string, cause error) *AppError {
	return New(QR_SERVICE_ERRORS, action+"_ERROR", message, cause)
}

func NewSystem(action, message string, cause error) *AppError {
	return New(SYSTEM_ERRORS, action+"_ERROR", message, cause)
}

func NewSocket(action, message string, cause error) *AppError {
	return New(SOCKET_ERRORS, action+"_ERROR", message, cause)
}

func NewOAuthService(action, message string, cause error) *AppError {
	return New(OAUTH_SERVICE_ERRORS, action+"_ERROR", message, cause)
}
