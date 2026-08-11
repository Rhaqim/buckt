package backend

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Rhaqim/buckt/internal/domain"
)

type MigrationBackendService struct {
	logger domain.BucktLogger

	primaryBackend   domain.FileBackend
	secondaryBackend domain.FileBackend

	// concurrency is how many files MigrateAll copies in parallel (>= 1).
	concurrency int

	migrating atomic.Bool
	stats     migrationStats
}

type migrationStats struct {
	mu        sync.Mutex
	completed int64 // files copied or already present in the secondary
	failed    int64 // files that permanently failed after retries
	total     int64
}

// defaultMigrationConcurrency mirrors buckt.DefaultMigrationConcurrency. It is
// duplicated here (rather than imported) because the root package imports this
// one, not the other way around.
const defaultMigrationConcurrency = 8

func NewMigrationBackend(bucktLogger domain.BucktLogger, primary domain.FileBackend, secondary domain.FileBackend, concurrency int) domain.MigratableBackend {
	if concurrency <= 0 {
		concurrency = defaultMigrationConcurrency
	}
	bucktLogger.Info("🚀 Initialising migration backend: " + primary.Name() + " -> " + secondary.Name())
	return &MigrationBackendService{
		logger:           bucktLogger,
		primaryBackend:   primary,
		secondaryBackend: secondary,
		concurrency:      concurrency,
	}
}

func (d *MigrationBackendService) Name() string {
	return d.primaryBackend.Name() + "->" + d.secondaryBackend.Name()
}

// Put writes to both backends. Primary failure is a hard error.
// Secondary failure is logged but does not fail the operation.
func (d *MigrationBackendService) Put(ctx context.Context, path string, data []byte) error {
	// Write to primary first — this is the source of truth
	if err := d.primaryBackend.Put(ctx, path, data); err != nil {
		return fmt.Errorf("primary backend put failed: %w", err)
	}

	// Mirror to secondary (best-effort during migration)
	if err := d.secondaryBackend.Put(ctx, path, data); err != nil {
		d.logger.Errorf("⚠️ Failed to mirror to secondary backend: %v", err)
	}
	return nil
}

// Get reads from primary (source of truth) first. If the primary has the
// file, it lazy-migrates forward to the secondary. If the primary is
// confirmed to not have the file (via Exists), it falls back to the secondary
// (assumes the file was already migrated forward and the primary cleaned up).
//
// Transient primary errors (network, permission, etc.) are propagated without
// falling back to secondary, so callers don't silently get stale data when
// the source of truth is temporarily unreachable.
//
// Lazy migration is always primary -> secondary; we never write back to the
// primary because that would risk overwriting authoritative data with stale
// secondary data.
func (d *MigrationBackendService) Get(ctx context.Context, path string) ([]byte, error) {
	// Fast path: try primary directly
	data, err := d.primaryBackend.Get(ctx, path)
	if err == nil {
		// Lazy forward migration: ensure secondary has a copy.
		// Best-effort — we don't fail the Get if mirroring fails.
		if exists, existsErr := d.secondaryBackend.Exists(ctx, path); existsErr == nil && !exists {
			if putErr := d.secondaryBackend.Put(ctx, path, data); putErr != nil {
				d.logger.Errorf("⚠️ Failed to lazy-migrate %s to secondary: %v", path, putErr)
			}
		}
		return data, nil
	}

	// Primary Get failed. Check whether it's a "not found" vs. a transient
	// error by asking the primary explicitly. If Exists says the file isn't
	// there, fall back to the secondary. Otherwise propagate the original
	// error so we don't serve stale data on transient failures.
	exists, existsErr := d.primaryBackend.Exists(ctx, path)
	if existsErr != nil {
		// Wrap the existsErr (the actual cause of the "unreachable"
		// classification) and include the original Get error for context.
		return nil, fmt.Errorf("primary backend unreachable for %s: %w (initial get error: %v)", path, existsErr, err)
	}
	if exists {
		// Primary has the file but Get still failed — this is a transient
		// error on the primary, not a missing file. Don't fall back.
		return nil, fmt.Errorf("primary backend get failed for %s: %w", path, err)
	}

	// Confirmed missing from primary — try secondary
	d.logger.Infof("Primary missing %s, falling back to secondary", path)
	data, err = d.secondaryBackend.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("both backends failed to get %s: %w", path, err)
	}
	return data, nil
}

// List lists from primary first, falls back to secondary.
func (d *MigrationBackendService) List(ctx context.Context, prefix string) ([]string, error) {
	paths, err := d.primaryBackend.List(ctx, prefix)
	if err != nil {
		d.logger.Errorf("Primary backend list failed, trying secondary: %v", err)
		paths, err = d.secondaryBackend.List(ctx, prefix)
		if err != nil {
			return nil, fmt.Errorf("both backends failed to list %s: %w", prefix, err)
		}
	}
	return paths, nil
}

// Stream streams from primary first, falls back to secondary.
func (d *MigrationBackendService) Stream(ctx context.Context, path string) (io.ReadCloser, error) {
	reader, err := d.primaryBackend.Stream(ctx, path)
	if err != nil {
		d.logger.Errorf("Primary backend stream failed, trying secondary: %v", err)
		reader, err = d.secondaryBackend.Stream(ctx, path)
		if err != nil {
			return nil, fmt.Errorf("both backends failed to stream %s: %w", path, err)
		}
	}
	return reader, nil
}

// Move moves in both backends.
func (d *MigrationBackendService) Move(ctx context.Context, oldPath string, newPath string) error {
	if err := d.primaryBackend.Move(ctx, oldPath, newPath); err != nil {
		return fmt.Errorf("primary backend move failed: %w", err)
	}
	if err := d.secondaryBackend.Move(ctx, oldPath, newPath); err != nil {
		d.logger.Errorf("⚠️ Failed to move in secondary backend: %v", err)
	}
	return nil
}

// Exists checks primary first, then secondary.
func (d *MigrationBackendService) Exists(ctx context.Context, path string) (bool, error) {
	exists, err := d.primaryBackend.Exists(ctx, path)
	if err != nil {
		d.logger.Errorf("Primary backend exists check failed, trying secondary: %v", err)
		exists, err = d.secondaryBackend.Exists(ctx, path)
		if err != nil {
			return false, fmt.Errorf("both backends failed exists check for %s: %w", path, err)
		}
	}
	return exists, nil
}

// Delete deletes from both backends. Primary failure is a hard error.
func (d *MigrationBackendService) Delete(ctx context.Context, path string) error {
	if err := d.primaryBackend.Delete(ctx, path); err != nil {
		return fmt.Errorf("primary backend delete failed: %w", err)
	}
	if err := d.secondaryBackend.Delete(ctx, path); err != nil {
		d.logger.Errorf("⚠️ Failed to delete from secondary backend: %v", err)
	}
	return nil
}

// DeleteFolder deletes from both backends.
func (d *MigrationBackendService) DeleteFolder(ctx context.Context, prefix string) error {
	if err := d.primaryBackend.DeleteFolder(ctx, prefix); err != nil {
		return fmt.Errorf("primary backend delete folder failed: %w", err)
	}
	if err := d.secondaryBackend.DeleteFolder(ctx, prefix); err != nil {
		d.logger.Errorf("⚠️ Failed to delete folder from secondary backend: %v", err)
	}
	return nil
}

const (
	// migrateMaxAttempts bounds how many times MigrateFile tries a single file
	// before giving up, so a transient primary/secondary hiccup doesn't drop the
	// file from the migration.
	migrateMaxAttempts = 3
	// migrateRetryBackoff is the base delay between attempts, scaled by the
	// attempt number (linear backoff).
	migrateRetryBackoff = 500 * time.Millisecond
)

// MigrateFile copies a single file from primary to secondary, retrying transient
// read/write failures with a linear backoff. On success it increments the
// completed counter exactly once; on exhaustion it returns the last error.
func (d *MigrationBackendService) MigrateFile(ctx context.Context, path string) error {
	var lastErr error
	for attempt := 1; attempt <= migrateMaxAttempts; attempt++ {
		if attempt > 1 {
			// Wait before retrying, but bail immediately if cancelled.
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(migrateRetryBackoff * time.Duration(attempt-1)):
			}
		}

		data, err := d.primaryBackend.Get(ctx, path)
		if err != nil {
			lastErr = fmt.Errorf("failed to read %s from primary: %w", path, err)
			continue
		}

		if err := d.secondaryBackend.Put(ctx, path, data); err != nil {
			lastErr = fmt.Errorf("failed to write %s to secondary: %w", path, err)
			continue
		}

		d.stats.mu.Lock()
		d.stats.completed++
		d.stats.mu.Unlock()

		return nil
	}

	return fmt.Errorf("migrate %s failed after %d attempts: %w", path, migrateMaxAttempts, lastErr)
}

// MigrateAll copies all files from primary to secondary in the background,
// using a bounded pool of d.concurrency workers. It is idempotent: objects
// already present in the secondary are skipped, so it is safe to re-run after
// an interruption.
func (d *MigrationBackendService) MigrateAll(ctx context.Context) error {
	// Atomic check-and-set so two concurrent callers can't both start a
	// migration. CompareAndSwap returns false if the value was already true.
	if !d.migrating.CompareAndSwap(false, true) {
		return fmt.Errorf("migration already in progress")
	}

	// List all files from primary
	files, err := d.primaryBackend.List(ctx, "")
	if err != nil {
		d.migrating.Store(false)
		return fmt.Errorf("failed to list files from primary: %w", err)
	}

	d.stats.mu.Lock()
	d.stats.total = int64(len(files))
	d.stats.completed = 0
	d.stats.mu.Unlock()

	// Build a skip-set from a single List of the secondary, instead of one
	// Exists round-trip per file (halving the request count on large buckets).
	// Keys are normalised with normaliseKey so both the leading-slash and
	// slash-stripped forms of a key match — see the cloud backends' altKey.
	skip, haveSkipSet := d.secondarySkipSet(ctx)

	// alreadyInSecondary reports whether the secondary already holds path. It
	// uses the pre-listed skip-set when available, and otherwise falls back to a
	// per-file Exists check (e.g. when the secondary's List failed).
	alreadyInSecondary := func(path string) bool {
		if haveSkipSet {
			_, ok := skip[normaliseKey(path)]
			return ok
		}
		exists, err := d.secondaryBackend.Exists(ctx, path)
		return err == nil && exists
	}

	workers := d.concurrency
	if workers <= 0 {
		workers = defaultMigrationConcurrency
	}
	if workers > len(files) {
		workers = len(files)
	}

	// Run migration in the background.
	go func() {
		defer d.migrating.Store(false)

		if len(files) == 0 {
			d.logger.Info("✅ Migration complete (nothing to copy)")
			return
		}

		paths := make(chan string)

		var wg sync.WaitGroup
		wg.Add(workers)
		for i := 0; i < workers; i++ {
			go func() {
				defer wg.Done()
				for path := range paths {
					if ctx.Err() != nil {
						return // cancelled — drain quietly
					}
					if alreadyInSecondary(path) {
						d.stats.mu.Lock()
						d.stats.completed++
						d.stats.mu.Unlock()
						continue
					}
					if err := d.MigrateFile(ctx, path); err != nil {
						d.logger.Errorf("Failed to migrate %s: %v", path, err)
						// Count it as processed-but-failed so progress still
						// reaches total (the badge completes) and callers get a
						// failure count. Continue with other files.
						d.stats.mu.Lock()
						d.stats.failed++
						d.stats.mu.Unlock()
					}
				}
			}()
		}

		// Feed paths to the workers, stopping early if the context is cancelled.
		cancelled := false
	feed:
		for _, path := range files {
			select {
			case <-ctx.Done():
				cancelled = true
				break feed
			case paths <- path:
			}
		}
		close(paths)
		wg.Wait()

		if cancelled || ctx.Err() != nil {
			d.logger.Errorf("Migration cancelled: %v", ctx.Err())
			return
		}
		d.logger.Info("✅ Migration complete")
	}()

	return nil
}

// secondarySkipSet lists the secondary once and returns the set of normalised
// keys it already holds. The bool is false when the listing failed, signalling
// callers to fall back to per-file Exists checks.
func (d *MigrationBackendService) secondarySkipSet(ctx context.Context) (map[string]struct{}, bool) {
	keys, err := d.secondaryBackend.List(ctx, "")
	if err != nil {
		d.logger.Errorf("Could not list secondary for skip-set (falling back to per-file checks): %v", err)
		return nil, false
	}
	set := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		set[normaliseKey(k)] = struct{}{}
	}
	return set, true
}

// normaliseKey strips a single leading slash so the two key forms buckt can
// produce ("/user/…" from nested-mode writes and "user/…" from a local List)
// compare equal.
func normaliseKey(key string) string {
	return strings.TrimPrefix(key, "/")
}

// MigrationStatus returns the current migration progress: files copied (or
// already present), files that permanently failed after retries, and the total
// scheduled. completed+failed == total once the run finishes.
func (d *MigrationBackendService) MigrationStatus(ctx context.Context) (completed int64, failed int64, total int64) {
	d.stats.mu.Lock()
	defer d.stats.mu.Unlock()
	return d.stats.completed, d.stats.failed, d.stats.total
}
