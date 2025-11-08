package migration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Rhaqim/buckt/internal/domain"
)

// MigrationBackendService manages dual-backend behaviour and migration.
type MigrationBackendService struct {
	logger domain.BucktLogger

	primary   domain.FileBackend
	secondary domain.FileBackend

	// migration
	mu            sync.RWMutex
	mode          MigrationMode
	active        atomic.Bool
	cfg           MigrationConfig
	checkpointMux sync.Mutex
	state         *migrationState
	statePath     string

	cancelMigration context.CancelFunc
	// wg              sync.WaitGroup
}

// NewMigrationBackend unchanged except returns *MigrationBackendService
/* Example usage:

var logger = domain.NewLogger()
var localBackend = domain.NewLocalFileBackend()
var s3Backend = domain.NewS3FileBackend()

var mgr = NewMigrationBackend(logger, localBackend, s3Backend)

mgr.EnableMigration(ctx, s3Backend, MigrateModeToSecondary, &MigrationConfig{
    Concurrency: 16,
    DeleteAfterCopy: false,
    PersistPath: "/var/run/bucket_migration_state.json",
})
go func() {
    err := mgr.MigrateTo(ctx, "images/", func(p string){ fmt.Println("migrated", p) }, func(p string, e error){ fmt.Println("err", p, e) })
    if err != nil {
       log.Println("migration finished with error", err)
    }
}()
defer func() {
	mgr.DisableMigration(ctx)
}()
*/
func NewMigrationBackend(logger domain.BucktLogger, primary domain.FileBackend, secondary domain.FileBackend) *MigrationBackendService {
	return &MigrationBackendService{
		logger:    logger,
		primary:   primary,
		secondary: secondary,
		mode:      MigrateModeNone,
	}
}

func (d *MigrationBackendService) EnableMigration(ctx context.Context, target domain.FileBackend, mode MigrationMode, cfg *MigrationConfig) error {
	if target == nil {
		return fmt.Errorf("target backend cannot be nil")
	}
	if cfg == nil {
		cfg = &MigrationConfig{}
	}
	// defaults
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 8
	}
	if cfg.RetryCount <= 0 {
		cfg.RetryCount = 3
	}
	if cfg.RetryBackoff == 0 {
		cfg.RetryBackoff = 500 * time.Millisecond
	}
	if cfg.PersistPath == "" {
		// simple default: use cwd/.migration_state.json
		cfg.PersistPath = ".migration_state.json"
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	d.secondary = target
	d.mode = mode
	d.cfg = *cfg
	d.statePath = cfg.PersistPath

	return nil
}

func (d *MigrationBackendService) DisableMigration(ctx context.Context) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.cancelMigration != nil {
		d.cancelMigration()
	}
	d.mode = MigrateModeNone
	d.secondary = nil
	d.active.Store(false)
}

func (d *MigrationBackendService) Put(ctx context.Context, path string, data []byte) error {
	d.mu.RLock()
	mode := d.mode
	secondary := d.secondary
	d.mu.RUnlock()

	switch mode {
	case MigrateModeToSecondary:
		// write to secondary primarily
		if err := secondary.Put(ctx, path, data); err != nil {
			// attempt fallback to primary if secondary fails
			d.logger.Errorf("secondary put failed: %v", err)
			if err2 := d.primary.Put(ctx, path, data); err2 != nil {
				d.logger.Errorf("primary fallback put also failed: %v", err2)
				return err2
			}
			return err
		}
		// optional: mirror to primary async (disabled by default)
		return nil
	case MigrateModeFromSecondary:
		// primary is main
		return d.primary.Put(ctx, path, data)
	default:
		return d.primary.Put(ctx, path, data)
	}
}

func (d *MigrationBackendService) Get(ctx context.Context, path string) ([]byte, error) {
	d.mu.RLock()
	mode := d.mode
	secondary := d.secondary
	d.mu.RUnlock()

	// If migrating to secondary, prefer secondary (new writes go there).
	if mode == MigrateModeToSecondary && secondary != nil {
		if data, err := secondary.Get(ctx, path); err == nil {
			return data, nil
		}
		// fallback to primary
	}
	// otherwise primary first
	if data, err := d.primary.Get(ctx, path); err == nil {
		return data, nil
	}
	if secondary != nil {
		return secondary.Get(ctx, path)
	}
	return nil, fmt.Errorf("not found")
}

func (d *MigrationBackendService) Delete(ctx context.Context, path string) error {
	// best-effort delete in both
	_ = d.primary.Delete(ctx, path)
	if d.secondary != nil {
		_ = d.secondary.Delete(ctx, path)
	}
	return nil
}

// func (d *MigrationBackendService) List(ctx context.Context, prefix string) ([]string, error) {
// 	d.mu.RLock()
// 	mode := d.mode
// 	secondary := d.secondary
// 	d.mu.RUnlock()

// 	// If migrating to secondary, prefer secondary (new writes go there).
// 	if mode == MigrateModeToSecondary && secondary != nil {
// 		if paths, err := secondary.List(ctx, prefix); err == nil {
// 			return paths, nil
// 		}
// 		// fallback to primary
// 	}
// 	// otherwise primary first
// 	if paths, err := d.primary.List(ctx, prefix); err == nil {
// 		return paths, nil
// 	}
// 	if secondary != nil {
// 		return secondary.List(ctx, prefix)
// 	}
// 	return nil, fmt.Errorf("not found")
// }

func (d *MigrationBackendService) loadState(prefix string) (*migrationState, error) {
	d.checkpointMux.Lock()
	defer d.checkpointMux.Unlock()
	// if file exists, read and unmarshal; else create new
	if _, err := os.Stat(d.statePath); err == nil {
		b, err := os.ReadFile(d.statePath)
		if err != nil {
			return nil, err
		}
		var st migrationState
		if err := json.Unmarshal(b, &st); err != nil {
			return nil, err
		}
		// if prefix changed, create new state
		if st.Prefix != prefix {
			st = migrationState{Prefix: prefix, Processed: map[string]bool{}, StartedAt: time.Now()}
		}
		return &st, nil
	}
	st := &migrationState{Prefix: prefix, Processed: map[string]bool{}, StartedAt: time.Now()}
	return st, nil
}

func (d *MigrationBackendService) persistState() error {
	d.checkpointMux.Lock()
	defer d.checkpointMux.Unlock()
	if d.state == nil {
		return nil
	}
	d.state.UpdatedAt = time.Now()
	b, err := json.MarshalIndent(d.state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(d.statePath, b, 0644)
}

func (d *MigrationBackendService) MigrateTo(ctx context.Context, prefix string, onProgress func(file string), onError func(file string, err error)) error {
	d.mu.RLock()
	secondary := d.secondary
	cfg := d.cfg
	d.mu.RUnlock()
	if secondary == nil {
		return errors.New("no secondary configured")
	}
	// ensure single migration at a time
	if d.active.Load() {
		return errors.New("migration already active")
	}

	// create cancellable ctx
	cctx, cancel := context.WithCancel(ctx)
	d.cancelMigration = cancel
	d.active.Store(true)
	defer func() {
		d.active.Store(false)
		cancel()
	}()

	// load checkpoint
	st, err := d.loadState(prefix)
	if err != nil {
		return err
	}
	d.state = st

	// list all objects under prefix using primary.List
	paths, err := d.primary.List(cctx, prefix)
	if err != nil {
		return err
	}
	d.state.Total = int64(len(paths))

	jobCh := make(chan string, 1024)
	errCh := make(chan error, 1)

	// spawn workers
	var wg sync.WaitGroup
	for i := 0; i < cfg.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-cctx.Done():
					return
				case p, ok := <-jobCh:
					if !ok {
						return
					}
					// skip if already processed
					if d.isProcessed(p) {
						continue
					}
					if err := d.migrateOneWithRetries(cctx, p, secondary, cfg.RetryCount, cfg.RetryBackoff, onError); err != nil {
						// report but continue
						if onError != nil {
							onError(p, err)
						}
						continue
					}
					// mark processed and persist
					d.markProcessed(p)
					if onProgress != nil {
						onProgress(p)
					}
					// optional delete source
					if cfg.DeleteAfterCopy {
						_ = d.primary.Delete(cctx, p)
					}
				}
			}
		}()
	}

	// feed jobs
FeedLoop:
	for _, p := range paths {
		select {
		case <-cctx.Done():
			break FeedLoop
		default:
		}
		if d.isProcessed(p) {
			atomic.AddInt64(&d.state.Completed, 1)
			continue
		}
		jobCh <- p
	}
	close(jobCh)
	close(errCh)

	// wait for workers
	wg.Wait()

	// persist final state
	_ = d.persistState()
	return nil
}

func (d *MigrationBackendService) isProcessed(path string) bool {
	d.checkpointMux.Lock()
	defer d.checkpointMux.Unlock()
	if d.state == nil || d.state.Processed == nil {
		return false
	}
	return d.state.Processed[path]
}

func (d *MigrationBackendService) markProcessed(path string) {
	d.checkpointMux.Lock()
	defer d.checkpointMux.Unlock()
	if d.state == nil {
		d.state = &migrationState{Processed: map[string]bool{}}
	}
	if d.state.Processed == nil {
		d.state.Processed = map[string]bool{}
	}
	if !d.state.Processed[path] {
		d.state.Processed[path] = true
		d.state.Completed++
	}
	// persist periodically (you may want to batch)
	_ = d.persistState()
}

func (d *MigrationBackendService) migrateOneWithRetries(ctx context.Context, path string, target domain.FileBackend, retries int, backoff time.Duration, onError func(string, error)) error {
	var last error
	for i := 0; i <= retries; i++ {
		if i > 0 {
			time.Sleep(backoff * time.Duration(i))
		}
		// read from primary
		data, err := d.primary.Get(ctx, path)
		if err != nil {
			last = err
			continue
		}
		// write to target
		if err := target.Put(ctx, path, data); err != nil {
			last = err
			continue
		}
		// success
		return nil
	}
	if onError != nil {
		onError(path, last)
	}
	return last
}

func (d *MigrationBackendService) MigrationStatus(ctx context.Context) (completed int64, total int64) {
	if d.state == nil {
		return 0, 0
	}
	return d.state.Completed, d.state.Total
}

func (d *MigrationBackendService) IsMigrating() bool {
	return d.active.Load()
}
