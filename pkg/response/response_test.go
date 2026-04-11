package response

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestSuccess(t *testing.T) {
	data := "test data"
	expected := APIResponse[string]{Data: data, Error: nil}
	result := Success(data)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Success() = %v, want %v", result, expected)
	}
}

func TestError(t *testing.T) {
	userMsg := "user error message"
	devMsg := "developer error message"
	expected := APIResponse[any]{
		Data: nil,
		Error: &APIError{
			Message: userMsg,
			Details: devMsg,
		},
	}
	result := Error(userMsg, devMsg)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Error() = %v, want %v", result, expected)
	}
}

func TestWrapError(t *testing.T) {
	userMsg := "user error message"
	err := errors.New("developer error message")

	expected := APIResponse[any]{
		Data: nil,
		Error: &APIError{
			Message: userMsg,
			Details: err.Error(),
		},
	}
	result := WrapError(userMsg, err)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("WrapError() = %v, want %v", result, expected)
	}

	// Test with nil error
	expected2 := Success[any](nil)
	result2 := WrapError(userMsg, nil)

	if !reflect.DeepEqual(result2, expected2) {
		t.Errorf("WrapError() with nil error = %v, want %v", result2, expected2)
	}
}

func TestWrapErrorSafe(t *testing.T) {
	userMsg := "user error message"
	err := errors.New("developer error message")

	// WrapErrorSafe should NOT expose internal error details
	expected := APIResponse[any]{
		Data: nil,
		Error: &APIError{
			Message: userMsg,
			Details: "",
		},
	}
	result := WrapErrorSafe(userMsg, err)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("WrapErrorSafe() = %v, want %v", result, expected)
	}
}

func TestWrapErrorUsageWithErr(t *testing.T) {
	userMsg := "user error message"
	devMsg := "developer error message"
	err := errors.New(devMsg)

	expected := APIResponse[any]{
		Data: nil,
		Error: &APIError{
			Message: userMsg,
			Details: devMsg,
		},
	}
	result := WrapError(userMsg, err)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("WrapError() = %v, want %v", result, expected)
	}
}

func TestWrapErrorUsageWithoutErr(t *testing.T) {
	userMsg := "user error message"
	result := WrapError(userMsg, nil)

	if result.Error != nil {
		t.Errorf("WrapError() = %v, want %v", result.Error, nil)
	}
}


func TestWrapErrorSafe_OmitsDetailsInJSON(t *testing.T) {
	result := WrapErrorSafe("user error", errors.New("internal secret"))

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	output := string(data)

	// The details field should be omitted entirely, not serialized as empty string
	if strings.Contains(output, "\"details\"") {
		t.Errorf("WrapErrorSafe JSON should not contain \"details\" field, got: %s", output)
	}
	if !strings.Contains(output, "user error") {
		t.Errorf("WrapErrorSafe JSON should contain user message, got: %s", output)
	}
	if strings.Contains(output, "internal secret") {
		t.Errorf("WrapErrorSafe JSON should not leak internal error message, got: %s", output)
	}
}

func TestWrapError_IncludesDetailsInJSON(t *testing.T) {
	result := WrapError("user error", errors.New("internal detail"))

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	output := string(data)

	if !strings.Contains(output, "internal detail") {
		t.Errorf("WrapError JSON should include details, got: %s", output)
	}
}
