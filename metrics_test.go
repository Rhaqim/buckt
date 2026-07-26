package buckt

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/Rhaqim/buckt/pkg/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMetricsTestClient(t *testing.T, m metrics.Recorder) *Client {
	t.Helper()
	dir := t.TempDir()
	sqlDB, err := sql.Open("sqlite3", filepath.Join(dir, "buckt.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	client, err := New(Config{
		DB:             DBConfig{Driver: SQLite, Database: sqlDB},
		MediaDir:       filepath.Join(dir, "media"),
		Log:            LogConfig{Silence: true},
		FlatNameSpaces: true,
		Metrics:        m,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestMetrics_BackendOpsRecorded(t *testing.T) {
	m := metrics.NewCollector()
	c := newMetricsTestClient(t, m)
	const user = "u1"

	// Upload → backend Put; Get → backend Get (or cache).
	id, err := c.UploadFile(user, "", "a.txt", "text/plain", []byte("hello"))
	require.NoError(t, err)
	_, err = c.GetFile(id)
	require.NoError(t, err)

	snap := m.Snapshot()
	local := snap["local"]
	require.NotNil(t, local, "expected metrics for the local backend, got: %v", snap)

	assert.GreaterOrEqual(t, local[metrics.OpPut].Count, int64(1), "at least one Put recorded")
	assert.Equal(t, int64(5), local[metrics.OpPut].Bytes, "Put byte count == len(data)")
	// The first GetFile is a cache miss, so it hits the backend Get at least once.
	assert.GreaterOrEqual(t, local[metrics.OpGet].Count, int64(1), "at least one Get recorded")

	// Every recorded op maps to an R2 billing class.
	for op := range local {
		assert.NotEmpty(t, metrics.R2Class(op), "op %q should classify into an R2 class", op)
	}
}

func TestMetrics_NilRecorderIsNoOp(t *testing.T) {
	// With no Metrics configured, uploads still work and nothing panics.
	c := newMetricsTestClient(t, nil)
	_, err := c.UploadFile("u1", "", "a.txt", "text/plain", []byte("hello"))
	require.NoError(t, err)
}

func TestClientStorageAndCacheStats(t *testing.T) {
	c := newMetricsTestClient(t, nil)
	const user = "u1"

	before, err := c.StorageBytes(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(0), before)

	_, err = c.UploadFile(user, "", "a.txt", "text/plain", []byte("hello")) // 5 bytes
	require.NoError(t, err)
	_, err = c.UploadFile(user, "", "b.txt", "text/plain", []byte("world!")) // 6 bytes
	require.NoError(t, err)

	after, err := c.StorageBytes(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(11), after, "StorageBytes should be the sum of file sizes")

	// CacheStats is readable (values depend on cache config; just exercise it).
	hits, misses := c.CacheStats()
	_ = hits
	_ = misses
}
