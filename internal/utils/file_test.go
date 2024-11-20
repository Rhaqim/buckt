package utils

import (
	"reflect"
	"testing"
)

func TestValidateFolderPath(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", []string{}},
		{"   ", []string{}},
		{"/valid/folder/path", []string{"valid", "folder", "path"}},
		{"/valid/folder/path/", []string{"valid", "folder", "path"}},
		{"valid/folder/path", []string{"valid", "folder", "path"}},
		{"valid//folder/path", []string{}},                          // Double slashes
		{"valid/folder/path/", []string{"valid", "folder", "path"}}, // Trailing slash
		{"///valid/folder/path///", []string{}},
		{"inva!id/folder/path", []string{}}, // Invalid character
		{"valid/123/folder", []string{"valid", "123", "folder"}},
		{"folder with spaces/valid", []string{"folder with spaces", "valid"}},
	}

	for _, test := range tests {
		result := ValidateFolderPath(test.input)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("For input '%s', expected %v but got %v", test.input, test.expected, result)
		}
	}
}

func TestIsValidFolderPath(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"valid/folder", true},
		{"valid123", true},
		{"inva!id/folder", false}, // Invalid character
		{"///valid///", true},
		{"folder with spaces", true},
		{"folder@", false}, // Invalid character
	}

	for _, test := range tests {
		result := isValidFolderPath(test.input)
		if result != test.expected {
			t.Errorf("For input '%s', expected %v but got %v", test.input, test.expected, result)
		}
	}
}
