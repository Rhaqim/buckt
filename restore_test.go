package buckt

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newNestedTestClient builds a real Client in nested-namespace mode (so restore
// exercises physical backend blob moves, not just DB path rewrites).
func newNestedTestClient(t *testing.T) *Client {
	t.Helper()
	dir := t.TempDir()

	sqlDB, err := sql.Open("sqlite3", filepath.Join(dir, "buckt.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	client, err := New(Config{
		DB:             DBConfig{Driver: SQLite, Database: sqlDB},
		MediaDir:       filepath.Join(dir, "media"),
		Log:            LogConfig{Silence: true},
		FlatNameSpaces: false,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestRestoreFileReturnsToOrigin(t *testing.T) {
	c := newNestedTestClient(t)
	const user = "u1"

	// root/Docs/report.txt
	docsID, err := c.NewFolder(user, "", "Docs", "")
	require.NoError(t, err)
	fileID, err := c.UploadFile(user, docsID, "report.txt", "text/plain", []byte("hello"))
	require.NoError(t, err)

	// Trash it — the origin (Docs) should be recorded.
	_, err = c.DeleteFile(fileID)
	require.NoError(t, err)

	trashed, err := c.GetFile(fileID)
	require.NoError(t, err)
	require.NotNil(t, trashed.OriginParentID, "trashing records the origin parent")
	assert.Equal(t, docsID, trashed.OriginParentID.String())

	// Restore — back under Docs, not root; origin cleared; blob followed.
	require.NoError(t, c.RestoreFile(user, fileID))

	restored, err := c.GetFile(fileID)
	require.NoError(t, err)
	assert.Equal(t, docsID, restored.ParentID.String(), "restored to original folder, not root")
	assert.Nil(t, restored.OriginParentID, "origin marker cleared after restore")
	assert.Equal(t, []byte("hello"), restored.Data, "blob physically moved to the restored path")

	docs, err := c.GetFolderWithContent(user, docsID)
	require.NoError(t, err)
	var found bool
	for _, f := range docs.Files {
		if f.ID.String() == fileID {
			found = true
		}
	}
	assert.True(t, found, "file is listed under Docs again")
}

func TestRestoreFileFallsBackToRootWhenOriginGone(t *testing.T) {
	c := newNestedTestClient(t)
	const user = "u1"

	docsID, err := c.NewFolder(user, "", "Docs", "")
	require.NoError(t, err)
	fileID, err := c.UploadFile(user, docsID, "report.txt", "text/plain", []byte("hi"))
	require.NoError(t, err)

	// Trash the file (origin = Docs), then permanently delete Docs.
	_, err = c.DeleteFile(fileID)
	require.NoError(t, err)
	_, err = c.DeleteFolderPermanently(user, docsID)
	require.NoError(t, err)

	// Restore — origin no longer exists, so it lands in root.
	require.NoError(t, c.RestoreFile(user, fileID))

	root, err := c.GetFolderWithContent(user, "")
	require.NoError(t, err)
	restored, err := c.GetFile(fileID)
	require.NoError(t, err)
	assert.Equal(t, root.ID.String(), restored.ParentID.String(), "fell back to root when origin was gone")
}

func TestRestoreFolderReturnsToOrigin(t *testing.T) {
	c := newNestedTestClient(t)
	const user = "u1"

	// root/Docs/Photos/pic.txt
	docsID, err := c.NewFolder(user, "", "Docs", "")
	require.NoError(t, err)
	photosID, err := c.NewFolder(user, docsID, "Photos", "")
	require.NoError(t, err)
	fileID, err := c.UploadFile(user, photosID, "pic.txt", "text/plain", []byte("img"))
	require.NoError(t, err)

	// Trash the Photos folder (with its nested file), then restore it.
	_, err = c.DeleteFolder(photosID)
	require.NoError(t, err)
	require.NoError(t, c.RestoreFolder(user, photosID))

	restored, err := c.GetFolderWithContent(user, photosID)
	require.NoError(t, err)
	require.NotNil(t, restored.ParentID)
	assert.Equal(t, docsID, restored.ParentID.String(), "folder restored under its original parent")

	// The nested file's blob followed the folder back out of trash.
	f, err := c.GetFile(fileID)
	require.NoError(t, err)
	assert.Equal(t, []byte("img"), f.Data)
}
