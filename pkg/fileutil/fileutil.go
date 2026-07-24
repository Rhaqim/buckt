package fileutil

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"strings"
)

// ProcessFile reads a multipart file header into memory and returns the filename and bytes.
func ProcessFile(file *multipart.FileHeader) (string, []byte, error) {
	return ProcessFileWithLimit(file, 0)
}

// ProcessFileWithLimit reads a multipart file with a size limit.
// If maxSize is 0, no limit is enforced.
func ProcessFileWithLimit(file *multipart.FileHeader, maxSize int64) (string, []byte, error) {
	// Reject early using the per-part size reported by multipart parsing
	// (this is the size of the form field, not the request's Content-Length)
	if maxSize > 0 && file.Size > maxSize {
		return "", nil, fmt.Errorf("file size %d exceeds maximum allowed %d bytes", file.Size, maxSize)
	}

	f, err := file.Open()
	if err != nil {
		return "", nil, err
	}
	defer func() { _ = f.Close() }()

	var reader io.Reader = f
	if maxSize > 0 {
		reader = io.LimitReader(f, maxSize+1) // +1 to detect overflow
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return "", nil, err
	}

	if maxSize > 0 && int64(len(data)) > maxSize {
		return "", nil, fmt.Errorf("file size exceeds maximum allowed %d bytes", maxSize)
	}

	return file.Filename, data, nil
}

// inlineSafeTypes is the allowlist of content types that are safe to render
// inline in a browser. Anything outside this set — most importantly text/html
// and image/svg+xml, both of which can execute script in the serving origin —
// is forced to download instead of render.
var inlineSafeTypes = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"image/gif":       true,
	"image/webp":      true,
	"image/bmp":       true,
	"image/x-icon":    true,
	"audio/mpeg":      true,
	"audio/ogg":       true,
	"audio/wav":       true,
	"audio/webm":      true,
	"video/mp4":       true,
	"video/webm":      true,
	"video/ogg":       true,
	"application/pdf": true,
	"text/plain":      true,
}

// InlineSafeContentType reports whether content of the given type may be served
// with Content-Disposition: inline without risking stored XSS in the serving
// origin. SVG and any HTML/XML/script type deliberately return false — they must
// be downloaded (attachment), not rendered.
//
// The check ignores any "; charset=..." parameter so "text/plain; charset=utf-8"
// still matches.
func InlineSafeContentType(contentType string) bool {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = strings.TrimSpace(ct[:i])
	}
	return inlineSafeTypes[ct]
}

// DispositionFor returns "inline" for content types on the inline-safe
// allowlist and "attachment" for everything else. Use it to decide the
// disposition when serving user-uploaded content that a browser will load
// same-origin.
func DispositionFor(contentType string) string {
	if InlineSafeContentType(contentType) {
		return "inline"
	}
	return "attachment"
}

// SafeContentDisposition builds a Content-Disposition header value that is
// safe against header injection. It sanitizes path separators from the
// filename and percent-encodes the result using RFC 5987 / 6266 filename*
// syntax so that non-ASCII characters are preserved without breaking the
// header.
//
// Example:
//
//	c.Header("Content-Disposition", fileutil.SafeContentDisposition("attachment", file.Name))
func SafeContentDisposition(disposition, filename string) string {
	// Sanitize path separators so the filename is always a single component
	filename = strings.ReplaceAll(filename, "/", "_")
	filename = strings.ReplaceAll(filename, "\\", "_")
	encoded := url.PathEscape(filename)
	return fmt.Sprintf("%s; filename*=UTF-8''%s", disposition, encoded)
}
