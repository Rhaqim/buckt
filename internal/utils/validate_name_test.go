package utils

import "testing"

func TestValidateFileName(t *testing.T) {
	valid := []string{
		"report.pdf",
		"report.2024.final.pdf",
		"a",
		"my file.txt",        // internal spaces are fine
		"image (1).png",      // parens are fine
		".gitignore",         // leading dot is a real, valid filename
		"UPPER_and-lower.JS", // mixed case / dash / underscore
	}
	for _, name := range valid {
		if err := ValidateFileName(name); err != nil {
			t.Errorf("expected %q to be valid, got: %v", name, err)
		}
	}

	invalid := []string{
		"",                                  // empty
		" leading.txt",                      // leading whitespace
		"trailing.txt ",                     // trailing whitespace
		".",                                 // current dir
		"..",                                // parent dir
		"../evil.txt",                       // traversal
		"../../victim/root_folder/evil.txt", // the H-1 exploit vector
		"a/b.txt",                           // forward-slash separator
		"a\\b.txt",                          // backslash separator
		"foo/..",                            // non-canonical
		"/abs.txt",                          // absolute-ish
	}
	for _, name := range invalid {
		if err := ValidateFileName(name); err == nil {
			t.Errorf("expected %q to be rejected, but it passed validation", name)
		}
	}
}

func TestValidateFolderName(t *testing.T) {
	const trash = "__trash__"

	valid := []string{"Photos", "my folder", "2024-reports", "a.b"}
	for _, name := range valid {
		if err := ValidateFolderName(name, trash); err != nil {
			t.Errorf("expected %q to be valid, got: %v", name, err)
		}
	}

	invalid := []string{
		"",              // empty
		" x",            // leading whitespace
		"x ",            // trailing whitespace
		trash,           // reserved
		".",             // current dir
		"..",            // parent dir
		"a/b",           // separator
		"a\\b",          // backslash
		"../escape",     // traversal
	}
	for _, name := range invalid {
		if err := ValidateFolderName(name, trash); err == nil {
			t.Errorf("expected %q to be rejected, but it passed validation", name)
		}
	}

	// Without a reserved list, the trash name is just an ordinary folder name.
	if err := ValidateFolderName(trash); err != nil {
		t.Errorf("expected %q to be valid when not reserved, got: %v", trash, err)
	}
}
