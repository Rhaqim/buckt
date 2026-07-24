package utils

import (
	"fmt"
	"io"
	"mime/multipart"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"

	"image"
	"image/jpeg"
	"image/png"
	"os"

	"github.com/nfnt/resize"
)

func GenerateThumbnail(inputPath, outputPath string, width uint) error {
	file, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	img, format, err := image.Decode(file)
	if err != nil {
		return err
	}

	// Resize image
	thumbnail := resize.Resize(width, 0, img, resize.Lanczos3)

	// Save the thumbnail
	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer func() { _ = outFile.Close() }()

	if format == "png" {
		return png.Encode(outFile, thumbnail)
	}
	return jpeg.Encode(outFile, thumbnail, nil)
}

func GenerateVideoPreview(inputPath, outputPath string) error {
	if err := validateExecPath(inputPath); err != nil {
		return fmt.Errorf("invalid input path: %w", err)
	}
	if err := validateExecPath(outputPath); err != nil {
		return fmt.Errorf("invalid output path: %w", err)
	}
	cmd := exec.Command("ffmpeg", "-i", inputPath, "-ss", "00:00:02", "-vframes", "1", outputPath)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to generate video preview: %v", err)
	}
	return nil
}

func GeneratePDFPreview(inputPath, outputPath string) error {
	if err := validateExecPath(inputPath); err != nil {
		return fmt.Errorf("invalid input path: %w", err)
	}
	if err := validateExecPath(outputPath); err != nil {
		return fmt.Errorf("invalid output path: %w", err)
	}
	cmd := exec.Command("convert", "-density", "150", inputPath+"[0]", "-quality", "90", outputPath)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to generate PDF preview: %v", err)
	}
	return nil
}

func ProcessFile(file *multipart.FileHeader) (string, []byte, error) {
	f, err := file.Open()
	if err != nil {
		return "", nil, err
	}
	defer func() { _ = f.Close() }()

	data, err := io.ReadAll(f)
	if err != nil {
		return "", nil, err
	}

	return file.Filename, data, nil
}

// ValidateFolderPath validates a folder path and splits it into components if valid.
func ValidateFolderPath(folderPath string) []string {
	// Return an empty list if the folder path is empty
	if len(strings.TrimSpace(folderPath)) == 0 {
		return []string{}
	}

	// Trim spaces
	folderPath = strings.TrimSpace(folderPath)

	// Check for invalid characters
	if !isValidFolderPath(folderPath) {
		return []string{}
	}

	// Ensure no double slashes
	if strings.Contains(folderPath, "//") {
		return []string{}
	}

	// Remove leading and trailing slashes
	folderPath = strings.Trim(folderPath, "/")

	// Split the folder path into components
	return strings.Split(folderPath, "/")
}

// isValidFolderPath checks if a folder path contains only valid characters (alphanumeric, spaces, slashes).
func isValidFolderPath(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '/' && r != ' ' && r != '-' {
			return false
		}
	}
	return true
}

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

// validateExecPath ensures a path is clean (== filepath.Clean(path)),
// absolute, and doesn't have a basename starting with a dash (which could
// be misinterpreted as a flag by external commands like ffmpeg or convert).
//
// Callers must pass a path that is already in canonical form. This function
// rejects non-canonical inputs rather than silently normalizing them so the
// validated path matches exactly what gets passed to exec.Command.
func validateExecPath(path string) error {
	if path != filepath.Clean(path) {
		return fmt.Errorf("path must be in canonical form: %s", path)
	}
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path must be absolute, got: %s", path)
	}
	if strings.HasPrefix(filepath.Base(path), "-") {
		return fmt.Errorf("path basename must not start with a dash: %s", path)
	}
	return nil
}
