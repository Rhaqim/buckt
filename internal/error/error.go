package error

import (
	"errors"
)

var (
	ErrFileAlreadyExists = errors.New("file already exists")
	ErrBucketNotFound    = errors.New("bucket not found")
)
