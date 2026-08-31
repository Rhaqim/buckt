// Package buckterr defines the sentinel errors buckt returns from its public
// API. Consumers branch on them with errors.Is to react to specific failures
// without depending on buckt's internal storage layer (GORM, the filesystem, or
// a cloud SDK):
//
//	id, err := client.UploadFile(user, parent, name, ct, data)
//	switch {
//	case errors.Is(err, buckterr.ErrInvalidName):
//		http.Error(w, "bad file name", http.StatusBadRequest)
//	case errors.Is(err, buckterr.ErrFileTooLarge):
//		http.Error(w, "file too large", http.StatusRequestEntityTooLarge)
//	case err != nil:
//		http.Error(w, "internal error", http.StatusInternalServerError)
//	}
//
// The same values are re-exported from the root buckt package (buckt.ErrNotFound
// == buckterr.ErrNotFound), so either import works with errors.Is.
//
// Errors are always wrapped with %w and carry a human-readable message; use
// errors.Is for control flow and err.Error() for logging.
package buckterr

import "errors"

var (
	// ErrNotFound is returned when a referenced file or folder does not exist.
	// Map it to HTTP 404.
	ErrNotFound = errors.New("not found")

	// ErrInvalidID is returned when an id is not a valid UUID. Map it to 400.
	ErrInvalidID = errors.New("invalid id")

	// ErrInvalidName is returned when a file or folder name is empty, contains
	// path separators or dot-segments, is non-canonical, or is reserved. Map it
	// to HTTP 400.
	ErrInvalidName = errors.New("invalid name")

	// ErrAlreadyExists is returned when a create/move/rename would collide with
	// an existing file or folder (unique constraint). Map it to HTTP 409.
	ErrAlreadyExists = errors.New("already exists")

	// ErrFileTooLarge is returned when an upload exceeds the configured
	// MaxFileSize. Map it to HTTP 413.
	ErrFileTooLarge = errors.New("file too large")

	// ErrPathTraversal is returned when a resolved storage path would escape the
	// backend's root. Map it to HTTP 400 (or 403).
	ErrPathTraversal = errors.New("path traversal detected")

	// ErrTrashBatchExceeded is returned when deleting a folder would move more
	// files than the configured MaxTrashBatchSize in one operation.
	ErrTrashBatchExceeded = errors.New("trash batch size exceeded")

	// ErrBackendUnavailable is returned when the storage backend cannot be
	// reached (as opposed to the object simply not existing).
	ErrBackendUnavailable = errors.New("backend unavailable")

	// ErrUploadRejected is returned when a configured upload scanner rejects a
	// file (e.g. malware detected, disallowed type). The scanner's own error is
	// wrapped alongside it, so errors.Is(err, ErrUploadRejected) matches while the
	// underlying reason remains inspectable. Map it to 4xx (e.g. 422). See
	// WithUploadScanner.
	ErrUploadRejected = errors.New("upload rejected")

	// ErrUnsupported is returned when an operation isn't supported by the active
	// backend — e.g. a presigned URL on the local filesystem backend, or during
	// migration mode. Map it to 501/400 as appropriate.
	ErrUnsupported = errors.New("operation not supported by this backend")
)
