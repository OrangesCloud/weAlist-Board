package response

// Error codes
const (
	ErrCodeNotFound      = "NOT_FOUND"
	ErrCodeAlreadyExists = "ALREADY_EXISTS"
	ErrCodeValidation    = "VALIDATION_ERROR"
	ErrCodeInternal      = "INTERNAL_ERROR"
	ErrCodeUnauthorized  = "UNAUTHORIZED"
	ErrCodeForbidden     = "FORBIDDEN"
)

// AppError represents a custom application error
type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	return e.Message
}

// NewNotFoundError creates a new not found error
func NewNotFoundError(message string, details string) *AppError {
	return &AppError{
		Code:    ErrCodeNotFound,
		Message: message,
		Details: details,
	}
}

// NewAlreadyExistsError creates a new already exists error
func NewAlreadyExistsError(message string, details string) *AppError {
	return &AppError{
		Code:    ErrCodeAlreadyExists,
		Message: message,
		Details: details,
	}
}

// NewValidationError creates a new validation error
func NewValidationError(message string, details string) *AppError {
	return &AppError{
		Code:    ErrCodeValidation,
		Message: message,
		Details: details,
	}
}

// NewInternalError creates a new internal error
func NewInternalError(message string, details string) *AppError {
	return &AppError{
		Code:    ErrCodeInternal,
		Message: message,
		Details: details,
	}
}

// NewUnauthorizedError creates a new unauthorized error
func NewUnauthorizedError(message string, details string) *AppError {
	return &AppError{
		Code:    ErrCodeUnauthorized,
		Message: message,
		Details: details,
	}
}

// NewForbiddenError creates a new forbidden error
func NewForbiddenError(message string, details string) *AppError {
	return &AppError{
		Code:    ErrCodeForbidden,
		Message: message,
		Details: details,
	}
}

// NewAppError creates a new application error with the given code, message, and details
func NewAppError(code string, message string, details string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Details: details,
	}
}
