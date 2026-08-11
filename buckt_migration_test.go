package buckt

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// memBackend is a trivial in-memory FileBackend used as a migration target, so
// the migration mechanism can be tested without a real cloud store.
type memBackend struct {
	mu   sync.Mutex
	objs map[string][]byte
}

func newMemBackend() *memBackend { return &memBackend{objs: map[string][]byte{}} }

func (m *memBackend) Name() string { return "mem" }

func (m *memBackend) Put(_ context.Context, path string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]byte, len(data))
	copy(cp, data)
	m.objs[path] = cp
	return nil
}

func (m *memBackend) Get(_ context.Context, path string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	d, ok := m.objs[path]
	if !ok {
		return nil, fmt.Errorf("mem: not found: %s", path)
	}
	return d, nil
}

func (m *memBackend) Stream(ctx context.Context, path string) (io.ReadCloser, error) {
	d, err := m.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(d)), nil
}

func (m *memBackend) Exists(_ context.Context, path string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.objs[path]
	return ok, nil
}

func (m *memBackend) Delete(_ context.Context, path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.objs, path)
	return nil
}

func (m *memBackend) DeleteFolder(_ context.Context, prefix string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k := range m.objs {
		if strings.HasPrefix(k, prefix) {
			delete(m.objs, k)
		}
	}
	return nil
}

func (m *memBackend) List(_ context.Context, prefix string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []string
	for k := range m.objs {
		if strings.HasPrefix(k, prefix) {
			out = append(out, k)
		}
	}
	return out, nil
}

func (m *memBackend) Move(_ context.Context, oldPath, newPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if d, ok := m.objs[oldPath]; ok {
		m.objs[newPath] = d
		delete(m.objs, oldPath)
	}
	return nil
}

func (m *memBackend) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.objs)
}

func TestMigration_BulkMigrateExistingFiles(t *testing.T) {
	dir := t.TempDir()
	mediaDir := filepath.Join(dir, "media")
	dbPath := filepath.Join(dir, "db.sqlite")
	const user = "u1"

	// Phase 1 — local only. The blobs land on the local disk and nowhere else,
	// modelling a deployment that started on the filesystem.
	db1, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	c1, err := New(Config{DB: DBConfig{Driver: SQLite, Database: db1}, MediaDir: mediaDir, Log: LogConfig{Silence: true}})
	require.NoError(t, err)
	_, err = c1.UploadFile(user, "", "a.txt", "text/plain", []byte("alpha"))
	require.NoError(t, err)
	_, err = c1.UploadFile(user, "", "b.txt", "text/plain", []byte("beta"))
	require.NoError(t, err)
	require.NoError(t, c1.Close())
	require.NoError(t, db1.Close())

	// Phase 2 — migration mode local -> target, reusing the same DB + media dir.
	target := newMemBackend()
	db2, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	c2, err := New(Config{
		DB:       DBConfig{Driver: SQLite, Database: db2},
		MediaDir: mediaDir,
		Log:      LogConfig{Silence: true},
		Backend:  BackendConfig{Source: LocalBackend(), Target: target, MigrationEnabled: true},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c2.Close(); _ = db2.Close() })

	// The backend name reflects both sides during migration (source->target),
	// which is what the UI's /backend badge surfaces.
	assert.Equal(t, "local->mem", c2.BackendName())

	_, total, ok := c2.MigrationStatus(context.Background())
	require.True(t, ok, "migration should be enabled")
	require.Equal(t, int64(0), total, "no migration run yet")
	require.Equal(t, 0, target.count(), "target starts empty")

	// Bulk-migrate the pre-existing local files to the target.
	require.NoError(t, c2.MigrateAll(context.Background()))
	require.Eventually(t, func() bool {
		done, tot, _ := c2.MigrationStatus(context.Background())
		return tot > 0 && done == tot
	}, 5*time.Second, 20*time.Millisecond, "migration should finish")

	done, tot, _ := c2.MigrationStatus(context.Background())
	assert.Equal(t, tot, done)
	assert.GreaterOrEqual(t, target.count(), 2, "both pre-existing files copied to the target")

	// A NEW upload in migration mode is mirrored to the target immediately
	// (dual-write), not left behind for the next MigrateAll.
	before := target.count()
	_, err = c2.UploadFile(user, "", "c.txt", "text/plain", []byte("gamma"))
	require.NoError(t, err)
	assert.Greater(t, target.count(), before, "new writes are dual-written to the target")
}

func TestMigration_NotEnabledReturnsError(t *testing.T) {
	c := newTestClient(t) // plain local backend

	assert.Equal(t, "local", c.BackendName(), "single-backend name")
	assert.ErrorIs(t, c.MigrateAll(context.Background()), ErrBackendUnavailable)

	_, _, ok := c.MigrationStatus(context.Background())
	assert.False(t, ok, "MigrationStatus reports not-enabled")
}
