package schema_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/Rhaqim/buckt/internal/database/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gormsqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormschema "gorm.io/gorm/schema"
)

// nopLogger satisfies schema.Logger without producing output during tests.
type nopLogger struct{}

func (nopLogger) Info(string)          {}
func (nopLogger) Infof(string, ...any) {}
func (nopLogger) Warn(string)          {}

func openGorm(t *testing.T) *gorm.DB { return openGormPrefixed(t, "") }

func openGormPrefixed(t *testing.T, prefix string) *gorm.DB {
	t.Helper()
	sqlDB, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	gdb, err := gorm.Open(
		gormsqlite.New(gormsqlite.Config{DriverName: "sqlite", Conn: sqlDB}),
		&gorm.Config{NamingStrategy: gormschema.NamingStrategy{TablePrefix: prefix}},
	)
	require.NoError(t, err)
	return gdb
}

func colExists(t *testing.T, gdb *gorm.DB, table, col string) bool {
	t.Helper()
	var n int64
	// pragma_table_info is a table-valued function; count matching columns.
	err := gdb.Raw(
		`SELECT count(*) FROM pragma_table_info(?) WHERE name = ?`, table, col,
	).Scan(&n).Error
	require.NoError(t, err)
	return n > 0
}

// Fixed, valid UUIDs for the seeded rows so IDs scan into uuid.UUID exactly as
// they would in a real database.
const (
	idRoot     = "11111111-1111-1111-1111-111111111111" // root_folder (live)
	idDocs     = "22222222-2222-2222-2222-222222222222" // Docs (live)
	idOldStuff = "33333333-3333-3333-3333-333333333333" // OldStuff (trashed folder)
	idFileA    = "44444444-4444-4444-4444-444444444444" // a.txt (live)
	idFileGone = "55555555-5555-5555-5555-555555555555" // gone.txt (trashed file)
)

// seedV141 builds a database shaped like the v1.4.1 release: folder_models and
// file_models both carry a deleted_at column, file_models has NO user_id, and
// some rows are soft-deleted (deleted_at set). This is exactly the state an
// existing production database is in before upgrading.
func seedV141(t *testing.T, gdb *gorm.DB) {
	t.Helper()

	require.NoError(t, gdb.Exec(`CREATE TABLE folder_models (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		parent_id TEXT,
		name TEXT NOT NULL,
		description TEXT,
		path TEXT NOT NULL UNIQUE,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`).Error)

	require.NoError(t, gdb.Exec(`CREATE TABLE file_models (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		path TEXT NOT NULL UNIQUE,
		content_type TEXT NOT NULL DEFAULT '',
		size INTEGER NOT NULL DEFAULT 0,
		parent_id TEXT NOT NULL,
		hash TEXT NOT NULL DEFAULT '',
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`).Error)

	// user u1: root -> Docs(live) -> a.txt(live); and OldStuff(trashed folder),
	// gone.txt(trashed file, but its parent Docs is live so it must be moved to
	// trash on its own).
	exec := func(q string, args ...any) { require.NoError(t, gdb.Exec(q, args...).Error) }

	exec(`INSERT INTO folder_models (id,user_id,parent_id,name,description,path,created_at,updated_at,deleted_at)
		VALUES (?,'u1',NULL,'root_folder','','/u1/root_folder','2024-01-01','2024-01-01',NULL)`, idRoot)
	exec(`INSERT INTO folder_models (id,user_id,parent_id,name,description,path,created_at,updated_at,deleted_at)
		VALUES (?,'u1',?,'Docs','','/u1/root_folder/Docs','2024-01-01','2024-01-01',NULL)`, idDocs, idRoot)
	exec(`INSERT INTO folder_models (id,user_id,parent_id,name,description,path,created_at,updated_at,deleted_at)
		VALUES (?,'u1',?,'OldStuff','','/u1/root_folder/OldStuff','2024-01-01','2024-01-01','2024-02-01')`, idOldStuff, idRoot)

	exec(`INSERT INTO file_models (id,name,path,content_type,size,parent_id,hash,created_at,updated_at,deleted_at)
		VALUES (?,'a.txt','/u1/root_folder/Docs/a.txt','text/plain',3,?,'h1','2024-01-01','2024-01-01',NULL)`, idFileA, idDocs)
	exec(`INSERT INTO file_models (id,name,path,content_type,size,parent_id,hash,created_at,updated_at,deleted_at)
		VALUES (?,'gone.txt','/u1/root_folder/Docs/gone.txt','text/plain',4,?,'h2','2024-01-01','2024-01-01','2024-02-01')`, idFileGone, idDocs)
}

func TestApply_PreservesLegacyTrashOnV141Upgrade(t *testing.T) {
	gdb := openGorm(t)
	seedV141(t, gdb)

	loader := schema.NewLoader(schema.DialectSQLite, nopLogger{}, "")
	require.NoError(t, loader.Apply(context.Background(), gdb))

	// Ledger recorded both migrations.
	var versions []int
	require.NoError(t, gdb.Raw(`SELECT version FROM buckt_schema_migrations ORDER BY version`).Scan(&versions).Error)
	assert.Equal(t, []int{1, 2, 3, 4, 5, 6}, versions)

	// deleted_at columns are gone from both tables.
	assert.False(t, colExists(t, gdb, "file_models", "deleted_at"), "file_models.deleted_at should be dropped")
	assert.False(t, colExists(t, gdb, "folder_models", "deleted_at"), "folder_models.deleted_at should be dropped")

	// NOTHING was deleted: all 3 folders and both files still exist.
	var folderCount, fileCount int64
	require.NoError(t, gdb.Raw(`SELECT count(*) FROM folder_models`).Scan(&folderCount).Error)
	require.NoError(t, gdb.Raw(`SELECT count(*) FROM file_models`).Scan(&fileCount).Error)
	assert.Equal(t, int64(4), folderCount, "3 seeded folders + 1 created __trash__")
	assert.Equal(t, int64(2), fileCount, "both files preserved, none purged")

	// A trash folder was created for u1.
	var trashID string
	require.NoError(t, gdb.Raw(
		`SELECT id FROM folder_models WHERE user_id='u1' AND name='__trash__' AND parent_id IS NULL`,
	).Scan(&trashID).Error)
	require.NotEmpty(t, trashID)

	// The soft-deleted folder and file now live under trash.
	var oldStuffParent, goneParent string
	require.NoError(t, gdb.Raw(`SELECT parent_id FROM folder_models WHERE id='33333333-3333-3333-3333-333333333333'`).Scan(&oldStuffParent).Error)
	require.NoError(t, gdb.Raw(`SELECT parent_id FROM file_models WHERE id='55555555-5555-5555-5555-555555555555'`).Scan(&goneParent).Error)
	assert.Equal(t, trashID, oldStuffParent, "trashed folder should be reparented into __trash__")
	assert.Equal(t, trashID, goneParent, "trashed file should be reparented into __trash__")

	// Live rows are untouched.
	var docsParent, aParent string
	require.NoError(t, gdb.Raw(`SELECT parent_id FROM folder_models WHERE id='22222222-2222-2222-2222-222222222222'`).Scan(&docsParent).Error)
	require.NoError(t, gdb.Raw(`SELECT parent_id FROM file_models WHERE id='44444444-4444-4444-4444-444444444444'`).Scan(&aParent).Error)
	assert.Equal(t, idRoot, docsParent)
	assert.Equal(t, idDocs, aParent)

	// file_models.user_id was backfilled from the parent folder.
	var x1User, x2User string
	require.NoError(t, gdb.Raw(`SELECT user_id FROM file_models WHERE id='44444444-4444-4444-4444-444444444444'`).Scan(&x1User).Error)
	require.NoError(t, gdb.Raw(`SELECT user_id FROM file_models WHERE id='55555555-5555-5555-5555-555555555555'`).Scan(&x2User).Error)
	assert.Equal(t, "u1", x1User)
	assert.Equal(t, "u1", x2User)

	// Blobs are still addressable: relocated rows kept their original Path.
	var gonePath string
	require.NoError(t, gdb.Raw(`SELECT path FROM file_models WHERE id='55555555-5555-5555-5555-555555555555'`).Scan(&gonePath).Error)
	assert.Equal(t, "/u1/root_folder/Docs/gone.txt", gonePath, "path preserved so the backend blob stays readable")
}

func TestApply_IsIdempotent(t *testing.T) {
	gdb := openGorm(t)
	seedV141(t, gdb)

	loader := schema.NewLoader(schema.DialectSQLite, nopLogger{}, "")
	require.NoError(t, loader.Apply(context.Background(), gdb))

	// Second Apply must be a no-op and must not error or move anything again.
	require.NoError(t, loader.Apply(context.Background(), gdb))

	var folderCount int64
	require.NoError(t, gdb.Raw(`SELECT count(*) FROM folder_models`).Scan(&folderCount).Error)
	assert.Equal(t, int64(4), folderCount, "no extra trash folder created on re-run")

	var versions []int
	require.NoError(t, gdb.Raw(`SELECT version FROM buckt_schema_migrations ORDER BY version`).Scan(&versions).Error)
	assert.Equal(t, []int{1, 2, 3, 4, 5, 6}, versions)
}

func TestApply_FreshDatabase(t *testing.T) {
	gdb := openGorm(t)

	loader := schema.NewLoader(schema.DialectSQLite, nopLogger{}, "")
	require.NoError(t, loader.Apply(context.Background(), gdb))

	// Tables exist, ledger is populated, and there's no leftover deleted_at.
	assert.True(t, colExists(t, gdb, "file_models", "user_id"))
	assert.False(t, colExists(t, gdb, "file_models", "deleted_at"))
	assert.False(t, colExists(t, gdb, "folder_models", "deleted_at"))

	var versions []int
	require.NoError(t, gdb.Raw(`SELECT version FROM buckt_schema_migrations ORDER BY version`).Scan(&versions).Error)
	assert.Equal(t, []int{1, 2, 3, 4, 5, 6}, versions)
}

func TestApply_WithTablePrefix(t *testing.T) {
	const prefix = "myapp_"
	gdb := openGormPrefixed(t, prefix)

	loader := schema.NewLoader(schema.DialectSQLite, nopLogger{}, prefix)
	require.NoError(t, loader.Apply(context.Background(), gdb))

	// Prefixed tables and a prefixed ledger were created.
	assert.True(t, colExists(t, gdb, prefix+"folder_models", "user_id"))
	assert.True(t, colExists(t, gdb, prefix+"file_models", "user_id"))

	var versions []int
	require.NoError(t, gdb.Raw(`SELECT version FROM `+prefix+`buckt_schema_migrations ORDER BY version`).Scan(&versions).Error)
	assert.Equal(t, []int{1, 2, 3, 4, 5, 6}, versions)

	// No un-prefixed tables were created.
	assert.False(t, colExists(t, gdb, "folder_models", "id"))
}

func TestApply_GuardsAgainstLegacyTablesWhenPrefixed(t *testing.T) {
	// A database with legacy un-prefixed tables (i.e. an existing v1.4.1 install)
	// must NOT be silently re-created empty under a new prefix.
	gdb := openGormPrefixed(t, "myapp_")
	seedV141(t, gdb) // creates un-prefixed folder_models / file_models

	loader := schema.NewLoader(schema.DialectSQLite, nopLogger{}, "myapp_")
	err := loader.Apply(context.Background(), gdb)
	require.Error(t, err, "must refuse to run a prefix over legacy un-prefixed tables")
	assert.Contains(t, err.Error(), "legacy un-prefixed tables")

	// The legacy data is untouched — no empty prefixed tables were created.
	assert.False(t, colExists(t, gdb, "myapp_folder_models", "id"))
	var folderCount int64
	require.NoError(t, gdb.Raw(`SELECT count(*) FROM folder_models`).Scan(&folderCount).Error)
	assert.Equal(t, int64(3), folderCount)
}
