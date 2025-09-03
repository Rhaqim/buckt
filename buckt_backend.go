package buckt

import (
	"fmt"
	"io"

	"github.com/Rhaqim/buckt/internal/backend"
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/pkg/logger"
)

// ResolveBackend picks the correct backend based on the config.
func resolveBackend(mediaDir string, bc BackendConfig, log *logger.BucktLogger, lru domain.LRUCache) Backend {
	switch {
	case bc.MigrationEnabled && bc.Source != nil && bc.Target != nil:
		log.Infof("🔄 Migration mode: %s → %s", bc.Source.Name(), bc.Target.Name())

		source := instantiateIfLocal(bc.Source, mediaDir, log, lru)
		target := instantiateIfLocal(bc.Target, mediaDir, log, lru)

		return backend.NewMigrationBackend(log, source, target)

	case bc.Source != nil:
		return instantiateIfLocal(bc.Source, mediaDir, log, lru)

	case bc.Target != nil:
		log.Warn("⚠️ Using target backend as primary because source is missing")
		return instantiateIfLocal(bc.Target, mediaDir, log, lru)

	default:
		log.Warn("⚠️ No backend configured, falling back to local")
		return backend.NewLocalFileSystemService(log, mediaDir, lru)
	}
}

func instantiateIfLocal(b Backend, mediaDir string, log *logger.BucktLogger, lru domain.LRUCache) Backend {
	if b.Name() == "local" {
		return backend.NewLocalFileSystemService(log, mediaDir, lru)
	}
	return b
}

type PlaceholderLocalBackend struct {
	name string
}

var _ domain.FileBackend = (*PlaceholderLocalBackend)(nil)

func (p *PlaceholderLocalBackend) Name() string { return p.name }

// Every other method should return an error, not panic
func (p *PlaceholderLocalBackend) Put(path string, data []byte) error {
	return fmt.Errorf("placeholder backend (%s) cannot be used directly", p.name)
}
func (p *PlaceholderLocalBackend) Get(path string) ([]byte, error) {
	return nil, fmt.Errorf("placeholder backend (%s) cannot be used directly", p.name)
}

// Delete implements domain.FileBackend.
func (p *PlaceholderLocalBackend) Delete(path string) error {
	return fmt.Errorf("placeholder backend (%s) cannot be used directly", p.name)
}

// DeleteFolder implements domain.FileBackend.
func (p *PlaceholderLocalBackend) DeleteFolder(prefix string) error {
	return fmt.Errorf("placeholder backend (%s) cannot be used directly", p.name)
}

// Exists implements domain.FileBackend.
func (p *PlaceholderLocalBackend) Exists(path string) (bool, error) {
	return false, fmt.Errorf("placeholder backend (%s) cannot be used directly", p.name)
}

// Move implements domain.FileBackend.
func (p *PlaceholderLocalBackend) Move(oldPath string, newPath string) error {
	return fmt.Errorf("placeholder backend (%s) cannot be used directly", p.name)
}

// Stream implements domain.FileBackend.
func (p *PlaceholderLocalBackend) Stream(path string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("placeholder backend (%s) cannot be used directly", p.name)
}
