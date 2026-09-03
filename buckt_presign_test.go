package buckt

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPresignedURL_UnsupportedOnLocal verifies that presigning is reported as
// unsupported (not a crash) on the local filesystem backend, which can't mint
// direct URLs. Cloud presigning is covered in cloud/aws (no network needed).
func TestPresignedURL_UnsupportedOnLocal(t *testing.T) {
	c := newTestClient(t) // local backend
	fileID, err := c.UploadFile("u1", "", "a.txt", "text/plain", []byte("x"))
	require.NoError(t, err)

	_, err = c.PresignedURL(fileID, 15*time.Minute)
	assert.ErrorIs(t, err, ErrUnsupported, "local backend cannot presign")

	_, err = c.PresignedDerivativeURL(fileID, "thumbnail", 15*time.Minute)
	assert.ErrorIs(t, err, ErrUnsupported)

	_, _, err = c.PresignUpload("u1", "", "b.txt", "text/plain", 15*time.Minute)
	assert.ErrorIs(t, err, ErrUnsupported, "local backend cannot presign uploads")
}

// presignClient builds a Client whose backend is a presign-capable in-memory
// backend (memBackend), so the register/confirm flow can run without a real
// cloud store.
func presignClient(t *testing.T) (*Client, *memBackend) {
	t.Helper()
	dir := t.TempDir()
	sqlDB, err := sql.Open("sqlite3", filepath.Join(dir, "b.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	target := newMemBackend()
	c, err := New(Config{
		DB:       DBConfig{Driver: SQLite, Database: sqlDB},
		MediaDir: filepath.Join(dir, "media"),
		Log:      LogConfig{Silence: true},
		Backend:  BackendConfig{Source: target},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Close() })
	return c, target
}

// TestPresignUpload_RegisterConfirmFlow drives the full direct-upload lifecycle:
// reserve (file ID valid, but hidden from listings) → client uploads straight to
// storage → finalize (file becomes live and readable).
func TestPresignUpload_RegisterConfirmFlow(t *testing.T) {
	c, target := presignClient(t)
	const user = "u1"
	ctx := context.Background()

	folderID, err := c.NewFolder(user, "", "Docs", "")
	require.NoError(t, err)

	// A normal file so the folder has one visible entry to compare against.
	normalID, err := c.UploadFile(user, folderID, "a.txt", "text/plain", []byte("a"))
	require.NoError(t, err)

	// Reserve a presigned upload.
	pendingID, url, err := c.PresignUpload(user, folderID, "b.pdf", "application/pdf", 15*time.Minute)
	require.NoError(t, err)
	require.NotEmpty(t, pendingID)
	require.True(t, strings.HasPrefix(url, "memput://"), "got a presigned PUT URL")

	// Before finalize the reserved file is hidden from listings.
	files, err := c.ListFiles(folderID)
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, normalID, files[0].ID.String(), "pending upload is not listed")

	// Simulate the client PUTting the bytes directly to storage.
	key := strings.TrimPrefix(url, "memput://")
	body := []byte("PDF-BYTES")
	require.NoError(t, target.Put(ctx, key, body))

	// Finalize — the file becomes live.
	gotID, err := c.FinalizeUpload(pendingID, int64(len(body)))
	require.NoError(t, err)
	assert.Equal(t, pendingID, gotID)

	// Now it's listed and readable, with the right size.
	files, err = c.ListFiles(folderID)
	require.NoError(t, err)
	assert.Len(t, files, 2, "finalized file now appears in listings")

	f, err := c.GetFile(pendingID)
	require.NoError(t, err)
	assert.Equal(t, body, f.Data)
	assert.False(t, f.Pending)
	assert.Equal(t, int64(len(body)), f.Size)
}

// TestFinalizeUpload_RejectsMissingObject verifies finalize fails if the client
// never actually uploaded the object (nothing at the reserved key).
func TestFinalizeUpload_RejectsMissingObject(t *testing.T) {
	c, _ := presignClient(t)
	pendingID, _, err := c.PresignUpload("u1", "", "never.pdf", "application/pdf", 15*time.Minute)
	require.NoError(t, err)

	_, err = c.FinalizeUpload(pendingID, 10)
	assert.ErrorIs(t, err, ErrNotFound, "finalize refuses a file whose object never landed")
}
