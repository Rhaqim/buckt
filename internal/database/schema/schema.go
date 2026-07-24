// Package schema provides buckt's versioned migration runner. It mirrors the
// structure of the loom engine's schema package — a dense, ordered list of
// migrations recorded in a ledger so each runs exactly once — but runs over a
// *gorm.DB instead of raw database/sql, so migrations can reuse GORM's
// AutoMigrate for buckt's model-defined baseline.
//
// Apply migrations with:
//
//	loader := schema.NewLoader(schema.DialectOf(db), log)
//	err := loader.Apply(ctx, db)
//
// Migrations live in migrations.go as a dense, ordered list. Each runs exactly
// once and is recorded in the buckt_schema_migrations ledger.
package schema

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// Dialect identifies the database flavour, matching loom's schema.Dialect.
type Dialect string

const (
	DialectPostgres Dialect = "postgres"
	DialectSQLite   Dialect = "sqlite"
)

// DialectOf maps a gorm dialector to a Dialect. Unknown dialectors fall back to
// SQLite semantics — the most conservative choice, since it avoids any
// Postgres-only DDL.
func DialectOf(db *gorm.DB) Dialect {
	if db.Dialector != nil && db.Dialector.Name() == "postgres" {
		return DialectPostgres
	}
	return DialectSQLite
}

// migrationsTable is the bare name of the ledger recording which versioned
// migrations have run. The active ledger name is this value with the loader's
// table prefix applied (see Loader.ledgerTable), so it tracks whichever schema
// the prefix selects.
const migrationsTable = "buckt_schema_migrations"

// Logger is the minimal logging surface the loader needs. domain.BucktLogger
// satisfies it, but the schema package depends only on this narrow interface so
// it stays decoupled from the rest of buckt.
type Logger interface {
	Info(message string)
	Infof(format string, args ...any)
	Warn(message string)
}

// Loader applies buckt's versioned schema migrations idempotently, tracked in a
// ledger so each runs exactly once. Mirrors loom's schema.Loader.
type Loader struct {
	dialect Dialect
	log     Logger
	prefix  string
}

// NewLoader creates a Loader for the given dialect. prefix is the table-name
// prefix that must match the gorm NamingStrategy prefix the database was opened
// with. The default (empty) prefix preserves buckt's historical table names
// (folder_models, file_models), so existing databases are adopted unchanged.
func NewLoader(dialect Dialect, log Logger, prefix string) *Loader {
	return &Loader{dialect: dialect, log: log, prefix: prefix}
}

// ledgerTable returns the prefixed name of the migrations ledger.
func (l *Loader) ledgerTable() string { return l.prefix + migrationsTable }

// Apply brings db up to the latest schema version. It is safe to call on every
// startup: already-applied migrations are skipped via the ledger, and the
// baseline adopts a pre-migration (v1.4.x) database in place — AutoMigrate only
// adds what is missing, so existing tables and data are preserved.
//
// Migrations run sequentially inside a transaction together with their ledger
// insert, so a failed migration rolls back cleanly and its version is never
// recorded. Single-instance startup is unaffected by concurrency; multi-instance
// coordination on a fresh database is a production concern (concurrent callers
// can race on the ledger insert, one failing with a duplicate version).
func (l *Loader) Apply(ctx context.Context, db *gorm.DB) error {
	if err := l.ensureLedger(db); err != nil {
		return fmt.Errorf("buckt schema: ledger: %w", err)
	}
	applied, err := l.appliedVersions(db)
	if err != nil {
		return fmt.Errorf("buckt schema: read ledger: %w", err)
	}
	for _, m := range migrations() {
		if applied[m.Version] {
			continue
		}
		l.log.Infof("🚀 applying schema migration %d (%s)...", m.Version, m.Name)
		if err := l.runMigration(ctx, db, m); err != nil {
			return fmt.Errorf("buckt schema: migration %d (%s): %w", m.Version, m.Name, err)
		}
		l.log.Info("✅ migration " + m.Name + " applied")
	}
	return nil
}

func (l *Loader) ensureLedger(db *gorm.DB) error {
	var stmt string
	if l.dialect == DialectPostgres {
		stmt = fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			version    INT PRIMARY KEY,
			name       TEXT NOT NULL DEFAULT '',
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`, l.ledgerTable())
	} else {
		stmt = fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			version    INTEGER PRIMARY KEY,
			name       TEXT NOT NULL DEFAULT '',
			applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`, l.ledgerTable())
	}
	return db.Exec(stmt).Error
}

func (l *Loader) appliedVersions(db *gorm.DB) (map[int]bool, error) {
	var versions []int
	if err := db.Raw(fmt.Sprintf(`SELECT version FROM %s`, l.ledgerTable())).Scan(&versions).Error; err != nil {
		return nil, err
	}
	out := make(map[int]bool, len(versions))
	for _, v := range versions {
		out[v] = true
	}
	return out, nil
}

// runMigration executes one migration and records it, both inside a single
// transaction so a failure rolls back the schema change and leaves the version
// unrecorded — exactly matching loom's runMigration semantics.
func (l *Loader) runMigration(ctx context.Context, db *gorm.DB, m Migration) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := m.Up(ctx, tx, l.prefix, l.dialect); err != nil {
			return err
		}
		return tx.Exec(
			fmt.Sprintf(`INSERT INTO %s (version, name) VALUES (?, ?)`, l.ledgerTable()),
			m.Version, m.Name,
		).Error
	})
}
