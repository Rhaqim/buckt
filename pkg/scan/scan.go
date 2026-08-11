// Package scan defines buckt's pluggable upload-scanning hook.
//
// A Scanner inspects a file's bytes on the upload path, before buckt commits
// them to the storage backend, and can reject the upload by returning an error.
// buckt intentionally ships no scanning engine of its own — bundling an
// antivirus or content classifier would drag heavy, deployment-specific
// dependencies into a storage library. Instead the consuming application
// supplies a Scanner: a ClamAV client, a VirusTotal lookup, a magic-byte /
// content-type allowlist, or any combination.
//
// Register one with buckt.WithUploadScanner. Because every upload path funnels
// through a single write chokepoint, a Scanner is guaranteed to see every file
// before it is stored — a caller cannot forget to wire scanning in per call
// site. A rejected upload never touches the backend and never fires a
// file.uploaded event; the failure surfaces as buckt.ErrUploadRejected wrapping
// the Scanner's own error.
package scan

import "context"

// Scanner inspects an upload before it is written to the backend.
//
// Scan is called synchronously on the upload path with the file's name and full
// bytes. Returning a non-nil error rejects the upload — the blob is not written,
// no metadata row survives, and no event fires. Returning nil admits it.
//
// Implementations must be safe for concurrent use: a single Scanner is shared
// across all uploads. Keep Scan reasonably fast, or front it with your own
// queue/worker, since it runs inline with the upload request.
type Scanner interface {
	Scan(ctx context.Context, name string, data []byte) error
}

// ScannerFunc adapts an ordinary function to a Scanner.
type ScannerFunc func(ctx context.Context, name string, data []byte) error

// Scan calls f(ctx, name, data).
func (f ScannerFunc) Scan(ctx context.Context, name string, data []byte) error {
	return f(ctx, name, data)
}
