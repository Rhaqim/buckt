package buckt

import (
	"github.com/Rhaqim/buckt/internal/backend"
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/pkg/logger"
)

// ResolveBackend picks the correct backend based on the config.
func resolveBackend(mediaDir string, bc BackendConfig, log *logger.BucktLogger, lru domain.LRUCache) Backend {

	switch {
	case bc.MigrationEnabled && bc.Source != nil && bc.Target != nil:
		log.Infof("🔄 Migration mode: %s → %s", bc.Source.Name(), bc.Target.Name())
		return backend.NewMigrationBackend(log, bc.Source, bc.Target)

	case bc.Source != nil:
		return bc.Source

	case bc.Target != nil:
		log.Warn("⚠️ Using target backend as primary because source is missing")
		return bc.Target

	default:
		log.Warn("⚠️ No backend configured, falling back to local")
		return backend.NewLocalFileSystemService(log, mediaDir, lru)
	}
}
