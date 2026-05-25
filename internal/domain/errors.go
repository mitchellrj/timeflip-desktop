package domain

import (
	"errors"
	"fmt"
	"strings"
)

type ErrorCode string

const (
	ErrDeviceNotFound              ErrorCode = "device_not_found"
	ErrBluetoothUnavailable        ErrorCode = "bluetooth_unavailable"
	ErrBluetoothPermissionDenied   ErrorCode = "bluetooth_permission_denied"
	ErrAuthorizationFailed         ErrorCode = "authorization_failed"
	ErrManualActionRequired        ErrorCode = "manual_action_required"
	ErrDeviceTimeout               ErrorCode = "device_timeout"
	ErrUnsupportedOperation        ErrorCode = "unsupported_operation"
	ErrConfigurationUnconfirmed    ErrorCode = "configuration_unconfirmed"
	ErrHistoryReconciliationFailed ErrorCode = "history_reconciliation_failed"
	ErrStorage                     ErrorCode = "storage_error"
	ErrValidation                  ErrorCode = "validation_failed"
)

var ErrNotFound = errors.New("not found")

type AppError struct {
	Code       ErrorCode `json:"code"`
	Message    string    `json:"message"`
	Diagnostic string    `json:"diagnostic,omitempty"`
	cause      error
}

func NewAppError(code ErrorCode, userMessage string, diagnosticMessage string, cause error) *AppError {
	return &AppError{
		Code:       code,
		Message:    userMessage,
		Diagnostic: RedactSensitive(diagnosticMessage),
		cause:      cause,
	}
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	if e.Diagnostic == "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Diagnostic)
}

func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

type DeviceWorkflowError struct{ *AppError }
type PersistenceError struct{ *AppError }
type ValidationError struct{ *AppError }
type TrackingError struct{ *AppError }

func RedactSensitive(value string) string {
	if value == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		"000000", "[REDACTED]",
		"password=", "password=[REDACTED]",
		"Password=", "Password=[REDACTED]",
	)
	return replacer.Replace(value)
}

func SafeLogFields(fields map[string]any) map[string]any {
	out := make(map[string]any, len(fields))
	for key, value := range fields {
		lower := strings.ToLower(key)
		if strings.Contains(lower, "password") || strings.Contains(lower, "payload") {
			out[key] = "[REDACTED]"
			continue
		}
		if s, ok := value.(string); ok {
			out[key] = RedactSensitive(s)
			continue
		}
		out[key] = value
	}
	return out
}
