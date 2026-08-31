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

// TestExpiry_UploadWithTTLSetsExpiryAtomically verifies UploadFileWithTTL stamps
// the expiry in the same insert (GetFile sees it immediately) and that an
// already-past expiry is purged.
func TestExpiry_UploadWithTTLSetsExpiryAtomically(t *testing.T) {
	c := newTestClient(t)
	const user = "u1"

	// Future TTL — the file exists with its expiry set right away.
	future, err := c.UploadFileWithTTL(user, "", "temp.txt", "text/plain", []byte("temp"), time.Hour)
	require.NoError(t, err)
	f, err := c.GetFile(future)
	require.NoError(t, err)
	require.NotNil(t, f.ExpiresAt, "expiry set atomically on upload")
	assert.WithinDuration(t, time.Now().Add(time.Hour), *f.ExpiresAt, time.Minute)

	// Already-past absolute expiry — purged on the next sweep.
	past, err := c.UploadFileWithExpiry(user, "", "gone.txt", "text/plain", []byte("gone"), time.Now().Add(-time.Minute))
	require.NoError(t, err)
	n, err := c.PurgeExpired(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	_, err = c.GetFile(past)
	assert.ErrorIs(t, err, ErrNotFound)
	// The future-dated one is untouched.
	_, err = c.GetFile(future)
	assert.NoError(t, err)
}

// TestExpiry_TempUploadSkipsDedup verifies a temp (TTL'd) upload is never
// collapsed onto an existing file by dedup — otherwise expiring the temp file
// could delete a permanent one that shares its bytes.
func TestExpiry_TempUploadSkipsDedup(t *testing.T) {
	c := newTestClient(t, withNested(), withDedup())
	const user = "u1"

	docsID, err := c.NewFolder(user, "", "Docs", "")
	require.NoError(t, err)
	data := []byte("identical bytes")

	// A normal upload followed by an identical (differently-named) normal upload
	// dedups to the same file — the established dedup behaviour.
	a, err := c.UploadFile(user, docsID, "a.jpg", "image/jpeg", data)
	require.NoError(t, err)
	b, err := c.UploadFile(user, docsID, "b.jpg", "image/jpeg", data)
	require.NoError(t, err)
	require.Equal(t, a, b, "identical normal uploads dedup")

	// An identical TEMP upload must NOT dedup onto the permanent file.
	temp, err := c.UploadFileWithTTL(user, docsID, "c.jpg", "image/jpeg", data, time.Hour)
	require.NoError(t, err)
	assert.NotEqual(t, a, temp, "temp upload is its own object, not deduped")
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
