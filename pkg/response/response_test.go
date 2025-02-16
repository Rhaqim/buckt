package response

import (
	"errors"
	"reflect"
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
	expected = Success[any](nil)
	result = WrapError(userMsg, nil)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("WrapError() with nil error = %v, want %v", result, expected)
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

	// asset that it fails
	if result.Error != nil {
		t.Errorf("WrapError() = %v, want %v", result.Error, nil)
	}
}
