package response

type APIResponse[T any] struct {
	Data  T         `json:"data,omitempty"`  // Success data
	Error *APIError `json:"error,omitempty"` // Error details (if any)
}

type APIError struct {
	Message string `json:"message"` // User-facing error message
	Details string `json:"details"` // Developer debug info (optional)
}

// Success response
func Success[T any](data T) APIResponse[T] {
	return APIResponse[T]{Data: data, Error: nil}
}

// User-friendly error response
func Error(userMsg, devMsg string) APIResponse[any] {
	return APIResponse[any]{
		Data: nil,
		Error: &APIError{
			Message: userMsg, // User-friendly message
			Details: devMsg,  // Internal developer message
		},
	}
}

// Wrap multiple errors together
func WrapError(userMsg string, err error) APIResponse[any] {
	if err == nil {
		return Success[any](nil)
	}
	return Error(userMsg, err.Error())
}
