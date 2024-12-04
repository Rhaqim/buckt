package error

import (
	"errors"
)

var (
	ErrFileAlreadyExists = errors.New("file already exists")
	ErrBucketNotFound    = errors.New("bucket not found")
	ErrInvalidUUID       = errors.New("invalid UUID")
	ErrFileNotFound      = errors.New("file not found")
	ErrFolderNotInPath   = errors.New("path must include at least one folder")
	ErrMinParentMinChild = errors.New("path must include at least one parent and one child folder")
	ErrFolderNotFound    = errors.New("folder not found")
)
