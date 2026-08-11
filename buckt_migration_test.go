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
	puts int // total Put calls, to detect wasteful re-uploads
	// failPuts makes the next N Put calls fail, to exercise retry behaviour.
	failPuts int
}

func newMemBackend() *memBackend { return &memBackend{objs: map[string][]byte{}} }

func (m *memBackend) Name() string { return "mem" }

func (m *memBackend) Put(_ context.Context, path string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.puts++
	if m.failPuts > 0 {
		m.failPuts--
		return fmt.Errorf("mem: transient put failure for %s", path)
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	m.objs[path] = cp
	return nil
}

func (m *memBackend) putCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.puts
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

// requireMigrationDone waits for the background copy to finish (completed==total).
func requireMigrationDone(t *testing.T, c *Client) {
	t.Helper()
	require.Eventually(t, func() bool {
		done, total, _ := c.MigrationStatus(context.Background())
		return total > 0 && done >= total
	}, 5*time.Second, 20*time.Millisecond, "migration should finish")
}

// TestMigration_MigrateAllIsIdempotent models an interrupted migration: the
// process stops and MigrateAll runs again on restart. Files already copied must
// NOT be re-uploaded — the target's Put count must not grow on the second pass.
func TestMigration_MigrateAllIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	mediaDir := filepath.Join(dir, "media")
	dbPath := filepath.Join(dir, "db.sqlite")
	const user = "u1"

	// Local-only files.
	db1, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	c1, err := New(Config{DB: DBConfig{Driver: SQLite, Database: db1}, MediaDir: mediaDir, Log: LogConfig{Silence: true}})
	require.NoError(t, err)
	for _, n := range []string{"a.txt", "b.txt", "c.txt"} {
		_, err := c1.UploadFile(user, "", n, "text/plain", []byte(n))
		require.NoError(t, err)
	}
	require.NoError(t, c1.Close())
	require.NoError(t, db1.Close())

	// Migration client.
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

	ctx := context.Background()

	// First pass copies everything.
	require.NoError(t, c2.MigrateAll(ctx))
	requireMigrationDone(t, c2)
	firstPuts := target.putCalls()
	objectsAfterFirst := target.count()
	require.GreaterOrEqual(t, firstPuts, 3, "all files copied on the first pass")

	// Second pass (as if the server restarted mid-migration and re-ran it).
	// Retry until the first goroutine has released the in-flight flag.
	require.Eventually(t, func() bool { return c2.MigrateAll(ctx) == nil }, 3*time.Second, 20*time.Millisecond)
	requireMigrationDone(t, c2)

	assert.Equal(t, firstPuts, target.putCalls(), "re-running migration must not re-upload already-copied files")
	assert.Equal(t, objectsAfterFirst, target.count(), "the object set is unchanged on the second pass")
}

// TestMigration_ConcurrentCopiesEachFileOnce drives MigrateAll with an explicit
// bounded worker pool over many files and asserts every file is copied exactly
// once (no double-uploads from a race, none skipped), threading the
// MigrationConcurrency config end-to-end.
func TestMigration_ConcurrentCopiesEachFileOnce(t *testing.T) {
	dir := t.TempDir()
	mediaDir := filepath.Join(dir, "media")
	dbPath := filepath.Join(dir, "db.sqlite")
	const user = "u1"
	const nFiles = 25

	db1, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	c1, err := New(Config{DB: DBConfig{Driver: SQLite, Database: db1}, MediaDir: mediaDir, Log: LogConfig{Silence: true}})
	require.NoError(t, err)
	for i := 0; i < nFiles; i++ {
		_, err := c1.UploadFile(user, "", fmt.Sprintf("file-%02d.txt", i), "text/plain", []byte(fmt.Sprintf("data-%02d", i)))
		require.NoError(t, err)
	}
	require.NoError(t, c1.Close())
	require.NoError(t, db1.Close())

	target := newMemBackend()
	db2, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	c2, err := New(Config{
		DB:       DBConfig{Driver: SQLite, Database: db2},
		MediaDir: mediaDir,
		Log:      LogConfig{Silence: true},
		Backend:  BackendConfig{Source: LocalBackend(), Target: target, MigrationEnabled: true, MigrationConcurrency: 4},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c2.Close(); _ = db2.Close() })

	require.NoError(t, c2.MigrateAll(context.Background()))
	requireMigrationDone(t, c2)

	assert.Equal(t, nFiles, target.count(), "every file copied to the target")
	assert.Equal(t, nFiles, target.putCalls(), "each file copied exactly once (no double-uploads)")
	done, total, _ := c2.MigrationStatus(context.Background())
	assert.Equal(t, int64(nFiles), total)
	assert.Equal(t, total, done)
}

// TestMigration_RetriesTransientFailure verifies MigrateFile retries a file
// whose first write fails transiently, so a flaky target doesn't drop files.
func TestMigration_RetriesTransientFailure(t *testing.T) {
	dir := t.TempDir()
	mediaDir := filepath.Join(dir, "media")
	dbPath := filepath.Join(dir, "db.sqlite")
	const user = "u1"

	db1, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	c1, err := New(Config{DB: DBConfig{Driver: SQLite, Database: db1}, MediaDir: mediaDir, Log: LogConfig{Silence: true}})
	require.NoError(t, err)
	_, err = c1.UploadFile(user, "", "only.txt", "text/plain", []byte("payload"))
	require.NoError(t, err)
	require.NoError(t, c1.Close())
	require.NoError(t, db1.Close())

	target := newMemBackend()
	target.failPuts = 1 // first copy attempt fails; the retry should succeed

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

	require.NoError(t, c2.MigrateAll(context.Background()))
	requireMigrationDone(t, c2)

	assert.Equal(t, 1, target.count(), "file lands despite the first attempt failing")
}

// TestMigration_PermanentFailureStillCompletes verifies that a file which fails
// every retry is counted as processed (so status reaches total and a progress
// badge doesn't hang) and surfaced via MigrationFailures.
func TestMigration_PermanentFailureStillCompletes(t *testing.T) {
	dir := t.TempDir()
	mediaDir := filepath.Join(dir, "media")
	dbPath := filepath.Join(dir, "db.sqlite")
	const user = "u1"

	db1, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	c1, err := New(Config{DB: DBConfig{Driver: SQLite, Database: db1}, MediaDir: mediaDir, Log: LogConfig{Silence: true}})
	require.NoError(t, err)
	_, err = c1.UploadFile(user, "", "doomed.txt", "text/plain", []byte("payload"))
	require.NoError(t, err)
	require.NoError(t, c1.Close())
	require.NoError(t, db1.Close())

	target := newMemBackend()
	target.failPuts = 100 // exceed any retry budget → permanent failure

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

	require.NoError(t, c2.MigrateAll(context.Background()))
	// requireMigrationDone asserts completed == total — it would hang (and fail
	// the timeout) if a permanently-failed file never counted as processed.
	requireMigrationDone(t, c2)

	failed, ok := c2.MigrationFailures(context.Background())
	require.True(t, ok)
	assert.Equal(t, int64(1), failed, "the un-copyable file is reported as failed")
	assert.Equal(t, 0, target.count(), "nothing landed in the target")
}

// TestMigration_ResumesFromPersistedState proves the DB-backed resume: after a
// migration records its progress, a restart (new client on the same DB) skips
// the already-copied files WITHOUT re-scanning the target. The second run uses a
// brand-new empty target that counts Put calls — a zero count proves the skip
// was driven by the persisted state, not by listing the target.
func TestMigration_ResumesFromPersistedState(t *testing.T) {
	dir := t.TempDir()
	mediaDir := filepath.Join(dir, "media")
	dbPath := filepath.Join(dir, "db.sqlite")
	const user = "u1"

	// Phase 1 — local-only files.
	db1, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	c1, err := New(Config{DB: DBConfig{Driver: SQLite, Database: db1}, MediaDir: mediaDir, Log: LogConfig{Silence: true}})
	require.NoError(t, err)
	for _, n := range []string{"a.txt", "b.txt", "c.txt"} {
		_, err := c1.UploadFile(user, "", n, "text/plain", []byte(n))
		require.NoError(t, err)
	}
	require.NoError(t, c1.Close())
	require.NoError(t, db1.Close())

	// Phase 2 — migrate to target1, which records the copies in the DB.
	target1 := newMemBackend()
	db2, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	c2, err := New(Config{
		DB:       DBConfig{Driver: SQLite, Database: db2},
		MediaDir: mediaDir,
		Log:      LogConfig{Silence: true},
		Backend:  BackendConfig{Source: LocalBackend(), Target: target1, MigrationEnabled: true},
	})
	require.NoError(t, err)
	require.NoError(t, c2.MigrateAll(context.Background()))
	requireMigrationDone(t, c2)
	require.Equal(t, 3, target1.count(), "first run copies everything")
	require.NoError(t, c2.Close())
	require.NoError(t, db2.Close())

	// Phase 3 — "restart": same DB, a fresh EMPTY target. Because the persisted
	// state already records all three keys as committed, MigrateAll must skip
	// them without a single Put to the new target.
	target2 := newMemBackend()
	db3, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	c3, err := New(Config{
		DB:       DBConfig{Driver: SQLite, Database: db3},
		MediaDir: mediaDir,
		Log:      LogConfig{Silence: true},
		Backend:  BackendConfig{Source: LocalBackend(), Target: target2, MigrationEnabled: true},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c3.Close(); _ = db3.Close() })

	require.NoError(t, c3.MigrateAll(context.Background()))
	requireMigrationDone(t, c3)

	assert.Equal(t, 0, target2.putCalls(), "resume skips already-migrated files without re-copying")
	done, total, _ := c3.MigrationStatus(context.Background())
	assert.Equal(t, int64(3), total)
	assert.Equal(t, total, done, "progress still reaches total on resume")
}

func TestMigration_NotEnabledReturnsError(t *testing.T) {
	c := newTestClient(t) // plain local backend

	assert.Equal(t, "local", c.BackendName(), "single-backend name")
	assert.ErrorIs(t, c.MigrateAll(context.Background()), ErrBackendUnavailable)

	_, _, ok := c.MigrationStatus(context.Background())
	assert.False(t, ok, "MigrationStatus reports not-enabled")
}
