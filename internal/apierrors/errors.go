// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package apierrors

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrorCode string

const (
	CodeInvalidRequest   ErrorCode = "REQ001"  // Invalid request params
	CodeUnauthorized     ErrorCode = "AUTH001" // Authentication required
	CodeForbidden        ErrorCode = "AUTH002" // Permission denied
	CodeDBAccessDenied   ErrorCode = "DB001"   // Database access denied
	CodeUserCredsReq      ErrorCode = "USR001" // DB credentials missing
	CodeUserCredsInvalid  ErrorCode = "USR002" // DB credentials invalid (rejected by DB)
	CodeUserCredsDecrypt  ErrorCode = "USR003" // Stored credentials could not be decrypted
	CodeNotFound          ErrorCode = "RES001" // Resource not found
	CodeInternalError    ErrorCode = "SYS001"  // Internal server error
)

type AppError struct {
	HTTPCode int       `json:"-"`
	Code     ErrorCode `json:"code"`
	Message  string    `json:"error"`
}

func (e *AppError) Error() string {
	return e.Message
}

// Predefined Errors
var (
	ErrInvalidRequest   = &AppError{http.StatusBadRequest, CodeInvalidRequest, "Invalid request parameters"}
	ErrUnauthorized     = &AppError{http.StatusUnauthorized, CodeUnauthorized, "Authentication required"}
	ErrForbidden        = &AppError{http.StatusForbidden, CodeForbidden, "You do not have permission to perform this action"}
	ErrDBAccessDenied   = &AppError{http.StatusForbidden, CodeDBAccessDenied, "Database access denied for this user"}
	ErrUserCredsReq      = &AppError{http.StatusForbidden, CodeUserCredsReq, "Database credentials required. Please provide your credentials."}
	ErrUserCredsInvalid  = &AppError{http.StatusForbidden, CodeUserCredsInvalid, "Database credentials are invalid. Please update your credentials."}
	ErrUserCredsDecrypt  = &AppError{http.StatusForbidden, CodeUserCredsDecrypt, "Stored credentials could not be decrypted. Please re-enter and save your database password."}
	ErrNotFound          = &AppError{http.StatusNotFound, CodeNotFound, "Resource not found"}
	ErrInternal         = &AppError{http.StatusInternalServerError, CodeInternalError, "An unexpected internal error occurred"}
)

// Helper function for Gin
func Respond(c *gin.Context, err error) {
	if appErr, ok := err.(*AppError); ok {
		c.JSON(appErr.HTTPCode, appErr)
		return
	}

	// For unexpected errors, log it (simplification here) and return a generic 500
	genericErr := &AppError{
		HTTPCode: http.StatusInternalServerError,
		Code:     CodeInternalError,
		Message:  err.Error(), // Exposing for dev config; in pure prod, this might just be "Internal Server Error"
	}
	c.JSON(genericErr.HTTPCode, genericErr)
}

// Helper to create a custom error dynamically
func NewAppError(httpCode int, code ErrorCode, message string) *AppError {
	return &AppError{
		HTTPCode: httpCode,
		Code:     code,
		Message:  message,
	}
}
