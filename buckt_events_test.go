package buckt

import (
	"context"
	"sync"
	"testing"

	"github.com/Rhaqim/buckt/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recorder is a thread-safe event sink for tests. Handlers run synchronously in
// these tests, but the mutex keeps -race happy and documents intent.
type recorder struct {
	mu   sync.Mutex
	seen []events.Event
}

func (r *recorder) handler() events.Handler {
	return func(_ context.Context, e events.Event) {
		r.mu.Lock()
		defer r.mu.Unlock()
		r.seen = append(r.seen, e)
	}
}

func (r *recorder) types() []events.Type {
	r.mu.Lock()
	defer r.mu.Unlock()
	ts := make([]events.Type, len(r.seen))
	for i, e := range r.seen {
		ts[i] = e.Type
	}
	return ts
}

func (r *recorder) last() events.Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.seen[len(r.seen)-1]
}

func TestEvents_FileLifecycle(t *testing.T) {
	var rec recorder
	c := newTestClient(t, withNested(), withEventHandler(rec.handler()))
	const user = "u1"

	// Upload → FileUploaded, with a populated payload.
	fileID, err := c.UploadFile(user, "", "a.txt", "text/plain", []byte("hello"))
	require.NoError(t, err)

	require.Equal(t, []events.Type{events.FileUploaded}, rec.types())
	up := rec.last()
	assert.Equal(t, fileID, up.FileID)
	assert.Equal(t, user, up.UserID)
	assert.Equal(t, "a.txt", up.Name)
	assert.Equal(t, "text/plain", up.ContentType)
	assert.Equal(t, int64(5), up.Size)
	assert.NotEmpty(t, up.Hash, "event carries the content hash")
	assert.False(t, up.Time.IsZero())

	// Trash → FileTrashed.
	_, err = c.DeleteFile(fileID)
	require.NoError(t, err)
	assert.Equal(t, events.FileTrashed, rec.last().Type)

	// Restore → FileRestored.
	require.NoError(t, c.RestoreFile(user, fileID))
	assert.Equal(t, events.FileRestored, rec.last().Type)

	// Permanent delete → FilePurged.
	_, err = c.DeleteFilePermanently(fileID)
	require.NoError(t, err)
	assert.Equal(t, events.FilePurged, rec.last().Type)

	assert.Equal(t,
		[]events.Type{events.FileUploaded, events.FileTrashed, events.FileRestored, events.FilePurged},
		rec.types(),
	)
}

func TestEvents_MultipleHandlersAndPanicIsolation(t *testing.T) {
	var a, b recorder
	// A panicking handler must not fail the upload nor stop other handlers.
	panicky := func(context.Context, events.Event) { panic("boom") }

	c := newTestClient(t,
		withEventHandler(a.handler()),
		withEventHandler(panicky),
		withEventHandler(b.handler()),
	)

	_, err := c.UploadFile("u1", "", "a.txt", "text/plain", []byte("hi"))
	require.NoError(t, err, "a panicking handler must not fail the operation")

	assert.Equal(t, []events.Type{events.FileUploaded}, a.types())
	assert.Equal(t, []events.Type{events.FileUploaded}, b.types(), "handlers after a panicking one still run")
}

func TestEvents_NoHandlerIsNoOp(t *testing.T) {
	c := newTestClient(t) // no handlers registered
	_, err := c.UploadFile("u1", "", "a.txt", "text/plain", []byte("hi"))
	require.NoError(t, err)
}
