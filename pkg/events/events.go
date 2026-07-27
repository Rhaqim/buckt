// Package events defines buckt's file lifecycle events and the handler
// signature applications register (via buckt.WithEventHandler) to react to
// them — for example, enqueue an uploaded image for AI processing or generate
// derivatives.
//
// buckt itself performs no processing; it only emits. This keeps buckt
// domain-agnostic and the dependency one-way: your worker imports buckt/events,
// never the reverse. Handlers run synchronously right after the operation
// succeeds (never inside the DB transaction), so keep them fast — the common
// pattern is to enqueue and return. A handler that panics or blocks affects
// only itself: panics are recovered and never fail the originating call.
package events

import (
	"context"
	"time"
)

// Type identifies a file lifecycle event.
type Type string

const (
	// FileUploaded fires after a new file's bytes are committed to the backend.
	FileUploaded Type = "file.uploaded"
	// FileTrashed fires after a file is moved to trash.
	FileTrashed Type = "file.trashed"
	// FileRestored fires after a file is restored from trash.
	FileRestored Type = "file.restored"
	// FilePurged fires after a file is permanently deleted.
	FilePurged Type = "file.purged"
)

// Event describes something that happened to a file. It carries only
// storage-level facts — no domain concepts — so handlers stay decoupled from
// both buckt's internals and your application model.
type Event struct {
	Type        Type      `json:"type"`
	UserID      string    `json:"user_id"`
	FileID      string    `json:"file_id"`
	ParentID    string    `json:"parent_id"`
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	Hash        string    `json:"hash"`
	Time        time.Time `json:"time"`
}

// Handler reacts to a buckt Event. Register handlers with buckt.WithEventHandler
// (it may be called more than once to attach several). The context is the one
// from the originating operation.
type Handler func(ctx context.Context, e Event)
