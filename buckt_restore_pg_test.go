package buckt

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRestoreReturnsToOrigin_Postgres mirrors the SQLite restore tests against
// Postgres in nested mode — the production configuration.
func TestRestoreReturnsToOrigin_Postgres(t *testing.T) {
	c := newTestClient(t, withPostgres(), withNested())
	const user = "u1"

	t.Run("file restores to origin folder", func(t *testing.T) {
		docsID, err := c.NewFolder(user, "", "Docs", "")
		require.NoError(t, err)
		fileID, err := c.UploadFile(user, docsID, "report.txt", "text/plain", []byte("hello"))
		require.NoError(t, err)

		_, err = c.DeleteFile(fileID)
		require.NoError(t, err)

		trashed, err := c.GetFile(fileID)
		require.NoError(t, err)
		require.NotNil(t, trashed.OriginParentID)
		assert.Equal(t, docsID, trashed.OriginParentID.String())

		require.NoError(t, c.RestoreFile(user, fileID))

		restored, err := c.GetFile(fileID)
		require.NoError(t, err)
		assert.Equal(t, docsID, restored.ParentID.String())
		assert.Nil(t, restored.OriginParentID)
		assert.Equal(t, []byte("hello"), restored.Data)
	})

	t.Run("non-empty folder trashes and restores to origin", func(t *testing.T) {
		docsID, err := c.NewFolder(user, "", "Projects", "")
		require.NoError(t, err)
		subID, err := c.NewFolder(user, docsID, "Alpha", "")
		require.NoError(t, err)
		fileID, err := c.UploadFile(user, subID, "notes.txt", "text/plain", []byte("data"))
		require.NoError(t, err)

		// This is the case that hit the pre-existing duplicate-key bug:
		// trashing a non-empty folder in nested mode.
		_, err = c.DeleteFolder(subID)
		require.NoError(t, err)
		require.NoError(t, c.RestoreFolder(user, subID))

		restored, err := c.GetFolderWithContent(user, subID)
		require.NoError(t, err)
		require.NotNil(t, restored.ParentID)
		assert.Equal(t, docsID, restored.ParentID.String())

		f, err := c.GetFile(fileID)
		require.NoError(t, err)
		assert.Equal(t, []byte("data"), f.Data)
	})
}
