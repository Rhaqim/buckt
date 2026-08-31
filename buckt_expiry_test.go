package buckt

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Rhaqim/buckt/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExpiry_PurgeExpiredDeletesExpiredOnly verifies PurgeExpired permanently
// deletes files past their expiry and leaves files without (or with a future)
// expiry untouched.
func TestExpiry_PurgeExpiredDeletesExpiredOnly(t *testing.T) {
	c := newTestClient(t)
	const user = "u1"

	expired, err := c.UploadFile(user, "", "temp.txt", "text/plain", []byte("temp"))
	require.NoError(t, err)
	keep, err := c.UploadFile(user, "", "keep.txt", "text/plain", []byte("keep"))
	require.NoError(t, err)

	// temp.txt is already past due; keep.txt never expires.
	require.NoError(t, c.SetFileExpiry(expired, time.Now().Add(-time.Minute)))

	n, err := c.PurgeExpired(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, n, "only the expired file is purged")

	_, err = c.GetFile(expired)
	assert.ErrorIs(t, err, ErrNotFound, "expired file is gone")

	_, err = c.GetFile(keep)
	assert.NoError(t, err, "non-expiring file survives")
}

// TestExpiry_FutureTTLNotPurgedAndClearable verifies a future TTL is not purged,
// and that clearing an expiry (zero time) makes the file permanent again.
func TestExpiry_FutureTTLNotPurgedAndClearable(t *testing.T) {
	c := newTestClient(t)
	fileID, err := c.UploadFile("u1", "", "a.txt", "text/plain", []byte("x"))
	require.NoError(t, err)

	// Expires in the future — a purge now is a no-op.
	require.NoError(t, c.SetFileTTL(fileID, time.Hour))
	n, err := c.PurgeExpired(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, n)
	_, err = c.GetFile(fileID)
	require.NoError(t, err)

	// Make it past-due, then clear the expiry — it must survive the next purge.
	require.NoError(t, c.SetFileExpiry(fileID, time.Now().Add(-time.Hour)))
	require.NoError(t, c.SetFileExpiry(fileID, time.Time{})) // zero time clears
	n, err = c.PurgeExpired(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, n)
	_, err = c.GetFile(fileID)
	require.NoError(t, err, "cleared expiry means the file is permanent again")
}

// TestExpiry_PurgeEmitsPurgedEvent verifies an expiry purge fires file.purged,
// so applications can hook post-deletion behaviour (the seam toward scheduled
// actions).
func TestExpiry_PurgeEmitsPurgedEvent(t *testing.T) {
	var mu sync.Mutex
	var purged []string
	h := func(_ context.Context, e events.Event) {
		if e.Type == events.FilePurged {
			mu.Lock()
			purged = append(purged, e.FileID)
			mu.Unlock()
		}
	}
	c := newTestClient(t, withEventHandler(h))

	fileID, err := c.UploadFile("u1", "", "t.txt", "text/plain", []byte("x"))
	require.NoError(t, err)
	require.NoError(t, c.SetFileExpiry(fileID, time.Now().Add(-time.Minute)))

	_, err = c.PurgeExpired(context.Background())
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	assert.Contains(t, purged, fileID, "file.purged fired for the expired file")
}

// TestExpiry_BackgroundSweeper verifies WithExpirySweeper auto-purges expired
// files on its interval, and that Close stops the goroutine.
func TestExpiry_BackgroundSweeper(t *testing.T) {
	dir := t.TempDir()
	sqlDB, err := sql.Open("sqlite3", filepath.Join(dir, "b.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	c, err := New(Config{
		DB:                  DBConfig{Driver: SQLite, Database: sqlDB},
		MediaDir:            filepath.Join(dir, "media"),
		Log:                 LogConfig{Silence: true},
		ExpirySweepInterval: 50 * time.Millisecond,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Close() })

	fileID, err := c.UploadFile("u1", "", "t.txt", "text/plain", []byte("x"))
	require.NoError(t, err)
	require.NoError(t, c.SetFileExpiry(fileID, time.Now().Add(-time.Minute)))

	require.Eventually(t, func() bool {
		_, err := c.GetFile(fileID)
		return errors.Is(err, ErrNotFound)
	}, 3*time.Second, 25*time.Millisecond, "background sweeper should purge the expired file")
}
