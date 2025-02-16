package utils

import (
	"fmt"
	"io"
	"mime/multipart"
	"os/exec"
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
	defer file.Close()

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
	defer outFile.Close()

	if format == "png" {
		return png.Encode(outFile, thumbnail)
	}
	return jpeg.Encode(outFile, thumbnail, nil)
}

func GenerateVideoPreview(inputPath, outputPath string) error {
	cmd := exec.Command("ffmpeg", "-i", inputPath, "-ss", "00:00:02", "-vframes", "1", outputPath)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to generate video preview: %v", err)
	}
	return nil
}

func GeneratePDFPreview(inputPath, outputPath string) error {
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
	defer f.Close()

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
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '/' || r == ' ' || r == '-') {
			return false
		}
	}
	return true
}
