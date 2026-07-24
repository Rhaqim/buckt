package buckt

import (
	"os"
	"path/filepath"
	"testing"

	"database/sql"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pgDSN returns the Postgres DSN from BUCKT_PG_DSN or skips the test.
func pgDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("BUCKT_PG_DSN")
	if dsn == "" {
		t.Skip("BUCKT_PG_DSN not set; skipping Postgres integration test")
	}
	return dsn
}

// newPGClient drops any existing buckt tables (fresh slate) and builds a real
// Client backed by the Postgres in BUCKT_PG_DSN.
func newPGClient(t *testing.T) *Client {
	t.Helper()
	sqlDB, err := sql.Open("postgres", pgDSN(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	for _, tbl := range []string{"file_models", "folder_models", "buckt_schema_migrations", "buckt_migration_models"} {
		_, err := sqlDB.Exec("DROP TABLE IF EXISTS " + tbl + " CASCADE")
		require.NoError(t, err)
	}

	client, err := New(Config{
		DB:             DBConfig{Driver: Postgres, Database: sqlDB},
		MediaDir:       filepath.Join(t.TempDir(), "media"),
		Log:            LogConfig{Silence: true},
		FlatNameSpaces: true,
		MaxFileSize:    1024,
	})
	require.NoError(t, err) // proves the migration runs cleanly on Postgres
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestClientSentinelErrors_Postgres(t *testing.T) {
	c := newPGClient(t)
	const user = "u1"

	t.Run("invalid id", func(t *testing.T) {
		_, err := c.GetFile("not-a-uuid")
		assert.ErrorIs(t, err, ErrInvalidID)
	})
	t.Run("not found", func(t *testing.T) {
		_, err := c.GetFile(uuid.NewString())
		assert.ErrorIs(t, err, ErrNotFound)
	})
	t.Run("invalid file name", func(t *testing.T) {
		_, err := c.UploadFile(user, "", "../evil.txt", "text/plain", []byte("hi"))
		assert.ErrorIs(t, err, ErrInvalidName)
	})
	t.Run("file too large", func(t *testing.T) {
		_, err := c.UploadFile(user, "", "big.bin", "application/octet-stream", make([]byte, 2048))
		assert.ErrorIs(t, err, ErrFileTooLarge)
	})
	t.Run("invalid folder name", func(t *testing.T) {
		_, err := c.NewFolder(user, "", "bad/name", "")
		assert.ErrorIs(t, err, ErrInvalidName)
	})
	t.Run("already exists (pg 23505 -> ErrAlreadyExists)", func(t *testing.T) {
		_, err := c.UploadFile(user, "", "dup.txt", "text/plain", []byte("one"))
		require.NoError(t, err)
		_, err = c.UploadFile(user, "", "dup.txt", "text/plain", []byte("two"))
		assert.ErrorIs(t, err, ErrAlreadyExists)
	})
}

// TestClientRoundTrip_Postgres exercises a real create/read/list/delete cycle on
// Postgres so the model queries (which the DeletedAt fix affects) are verified
// against the real dialect, not just SQLite.
func TestClientRoundTrip_Postgres(t *testing.T) {
	c := newPGClient(t)
	const user = "u1"

	id, err := c.UploadFile(user, "", "hello.txt", "text/plain", []byte("hello world"))
	require.NoError(t, err)
	require.NotEmpty(t, id)

	f, err := c.GetFile(id)
	require.NoError(t, err)
	assert.Equal(t, "hello.txt", f.Name)
	assert.Equal(t, []byte("hello world"), f.Data)
	assert.False(t, f.DeletedAt.Valid) //nolint:staticcheck // intentionally verifying the deprecated compat field is present and never populated

	root, err := c.GetFolderWithContent(user, "")
	require.NoError(t, err)
	require.NotNil(t, root)

	_, err = c.DeleteFile(id) // moves to trash
	require.NoError(t, err)
}
