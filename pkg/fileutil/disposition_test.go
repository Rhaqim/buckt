package fileutil

import "testing"

func TestDispositionFor(t *testing.T) {
	inline := []string{
		"image/png",
		"image/jpeg",
		"application/pdf",
		"video/mp4",
		"text/plain",
		"text/plain; charset=utf-8", // charset param ignored
		"IMAGE/PNG",                 // case-insensitive
	}
	for _, ct := range inline {
		if got := DispositionFor(ct); got != "inline" {
			t.Errorf("DispositionFor(%q) = %q, want inline", ct, got)
		}
	}

	attachment := []string{
		"text/html",                  // XSS vector
		"image/svg+xml",              // scriptable image
		"application/xhtml+xml",      // scriptable
		"application/octet-stream",   // unknown binary
		"application/javascript",     // script
		"",                           // empty
		"text/html; charset=utf-8",   // param does not make it safe
	}
	for _, ct := range attachment {
		if got := DispositionFor(ct); got != "attachment" {
			t.Errorf("DispositionFor(%q) = %q, want attachment", ct, got)
		}
	}
}
