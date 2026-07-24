package response

type APIResponse[T any] struct {
	Data  T         `json:"data,omitempty"`  // Success data
	Error *APIError `json:"error,omitempty"` // Error details (if any)
}

type APIError struct {
	Message string `json:"message"`           // User-facing error message
	Details string `json:"details,omitempty"` // Developer debug info (omitted when empty)
}

// Success response
func Success[T any](data T) APIResponse[T] {
	return APIResponse[T]{Data: data, Error: nil}
}

// Error builds an error response with both a user-facing message and
// optional developer details. Pass an empty string for devMsg to omit the
// details field from the JSON output entirely.
func Error(userMsg, devMsg string) APIResponse[any] {
	return APIResponse[any]{
		Data: nil,
		Error: &APIError{
			Message: userMsg,
			Details: devMsg,
		},
	}
}

// WrapError wraps an error into an API response, including the error details.
func WrapError(userMsg string, err error) APIResponse[any] {
	if err == nil {
		return Success[any](nil)
	}
	return Error(userMsg, err.Error())
}

// WrapErrorSafe wraps an error but omits the internal details from the
// response. Use this for public-facing APIs where internal errors should not
// be exposed. The "details" field is omitted from the JSON output entirely.
func WrapErrorSafe(userMsg string, err error) APIResponse[any] {
	if err == nil {
		return Success[any](nil)
	}
	return Error(userMsg, "")
}
