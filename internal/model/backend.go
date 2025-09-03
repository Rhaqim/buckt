package model

import "time"

type FileInfo struct {
	Size         int64
	LastModified time.Time
	ETag         string
	ContentType  string
}
