package buckt

import "github.com/Rhaqim/buckt/pkg/buckterr"

// Sentinel errors returned by Client methods. They are re-exported from
// pkg/buckterr so callers can branch on failures with errors.Is without an extra
// import:
//
//	if errors.Is(err, buckt.ErrNotFound) { ... }
//
// Because these are the same values as buckterr.ErrNotFound etc., either package
// works with errors.Is. See pkg/buckterr for the full documentation and
// suggested HTTP status mappings.
var (
	ErrNotFound           = buckterr.ErrNotFound
	ErrInvalidID          = buckterr.ErrInvalidID
	ErrInvalidName        = buckterr.ErrInvalidName
	ErrAlreadyExists      = buckterr.ErrAlreadyExists
	ErrFileTooLarge       = buckterr.ErrFileTooLarge
	ErrPathTraversal      = buckterr.ErrPathTraversal
	ErrTrashBatchExceeded = buckterr.ErrTrashBatchExceeded
	ErrBackendUnavailable = buckterr.ErrBackendUnavailable
	ErrUploadRejected     = buckterr.ErrUploadRejected
)
