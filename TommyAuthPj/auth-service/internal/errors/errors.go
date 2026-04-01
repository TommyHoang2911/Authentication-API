package apperrors

import "fmt"

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
	return fmt.Sprintf("%s: %v", e.Message, e.Cause)
}

func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func New(code, message string, cause error) *AppError {
	return &AppError{Code: code, Message: message, Cause: cause}
}

func NewValidation(message string) *AppError {
	return New("validation_error", message, nil)
}

func NewConfiguration(message string, cause error) *AppError {
	return New("configuration_error", message, cause)
}
