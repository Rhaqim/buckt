package error

import (
	"errors"
)

var (
	ErrInvalidUUID    = errors.New("invalid UUID")
	ErrFileNotFound   = errors.New("file not found")
	ErrFolderNotFound = errors.New("folder not found")
)
