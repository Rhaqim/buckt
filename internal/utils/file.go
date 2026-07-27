package utils

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ValidateFileName rejects names that would escape their parent folder when
// joined to a path, or be written verbatim as a cross-tenant object key on a
// cloud backend. A valid file name is a single, canonical path component.
//
// Dots are permitted so ordinary extensions ("report.2024.pdf", ".gitignore")
// are valid; only the whole-name dot-segments "." and ".." are rejected.
//
// Enforced in the file service's CreateFile, UpdateFile and RenameFile — the
// three places a caller-supplied name reaches filepath.Join(parent, name) and
// then the storage backend.
func ValidateFileName(name string) error {
	if name == "" {
		return fmt.Errorf("file name cannot be empty")
	}
	// Leading/trailing whitespace slips through multipart uploads and breaks
	// uniqueness checks in surprising ways.
	if name != strings.TrimSpace(name) {
		return fmt.Errorf("file name cannot have leading or trailing whitespace")
	}
	if strings.ContainsAny(name, "/\\") {
		return fmt.Errorf("file name cannot contain path separators")
	}
	// "." collapses to the parent and ".." escapes upward when joined.
	if name == "." || name == ".." {
		return fmt.Errorf("%q is not a valid file name", name)
	}
	// filepath.Base strips any directory portion; if it differs the name carried
	// separators or path structure. filepath.Clean catches non-canonical forms.
	if filepath.Base(name) != name || filepath.Clean(name) != name {
		return fmt.Errorf("file name must be a single canonical path component: %q", name)
	}
	return nil
}

// ValidateFolderName rejects names that are empty, contain path separators,
// resolve to a different path component when joined to a parent (e.g. ".",
// ".."), carry leading/trailing whitespace, are non-canonical, or collide with
// any of the caller-supplied reserved names (e.g. the trash folder).
//
// Reserved names are passed in by the caller so this package needs no
// dependency on internal/constant.
func ValidateFolderName(name string, reserved ...string) error {
	if name == "" {
		return fmt.Errorf("folder name cannot be empty")
	}
	if name != strings.TrimSpace(name) {
		return fmt.Errorf("folder name cannot have leading or trailing whitespace")
	}
	for _, r := range reserved {
		if name == r {
			return fmt.Errorf("%q is a reserved folder name", name)
		}
	}
	if strings.ContainsAny(name, "/\\") {
		return fmt.Errorf("folder name cannot contain path separators")
	}
	if name == "." || name == ".." {
		return fmt.Errorf("%q is not a valid folder name", name)
	}
	if filepath.Clean(name) != name {
		return fmt.Errorf("folder name must be in canonical form: %q", name)
	}
	return nil
}
