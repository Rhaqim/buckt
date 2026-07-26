package schema_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/Rhaqim/buckt/internal/database/schema"
	_ "github.com/lib/pq" // registers the "postgres" database/sql driver
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gormpg "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// pgGorm opens a gorm connection to BUCKT_PG_DSN (or skips), dropping any
// existing buckt tables so each run starts clean.
func pgGorm(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("BUCKT_PG_DSN")
	if dsn == "" {
		t.Skip("BUCKT_PG_DSN not set; skipping Postgres schema test")
	}
	sqlDB, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	gdb, err := gorm.Open(gormpg.New(gormpg.Config{DriverName: "postgres", Conn: sqlDB}), &gorm.Config{})
	require.NoError(t, err)
	for _, tbl := range []string{"file_models", "folder_models", "buckt_schema_migrations"} {
		require.NoError(t, gdb.Exec("DROP TABLE IF EXISTS "+tbl+" CASCADE").Error)
	}
	return gdb
}

// seedV141PG builds a v1.4.1-shaped Postgres schema (uuid pks, timestamptz
// deleted_at, no user_id on files) with the same rows as the SQLite fixture.
func seedV141PG(t *testing.T, gdb *gorm.DB) {
	t.Helper()
	exec := func(q string, args ...any) { require.NoError(t, gdb.Exec(q, args...).Error) }

	exec(`CREATE TABLE folder_models (
		id uuid PRIMARY KEY,
		user_id text NOT NULL,
		parent_id uuid,
		name text NOT NULL,
		description text,
		path text NOT NULL UNIQUE,
		created_at timestamptz,
		updated_at timestamptz,
		deleted_at timestamptz
	)`)
	exec(`CREATE TABLE file_models (
		id uuid PRIMARY KEY,
		name text NOT NULL,
		path text NOT NULL UNIQUE,
		content_type text NOT NULL DEFAULT '',
		size bigint NOT NULL DEFAULT 0,
		parent_id uuid NOT NULL,
		hash text NOT NULL DEFAULT '',
		created_at timestamptz,
		updated_at timestamptz,
		deleted_at timestamptz
	)`)

	exec(`INSERT INTO folder_models (id,user_id,parent_id,name,description,path,created_at,updated_at,deleted_at)
		VALUES (?::uuid,'u1',NULL,'root_folder','','/u1/root_folder',now(),now(),NULL)`, idRoot)
	exec(`INSERT INTO folder_models (id,user_id,parent_id,name,description,path,created_at,updated_at,deleted_at)
		VALUES (?::uuid,'u1',?::uuid,'Docs','','/u1/root_folder/Docs',now(),now(),NULL)`, idDocs, idRoot)
	exec(`INSERT INTO folder_models (id,user_id,parent_id,name,description,path,created_at,updated_at,deleted_at)
		VALUES (?::uuid,'u1',?::uuid,'OldStuff','','/u1/root_folder/OldStuff',now(),now(),now())`, idOldStuff, idRoot)

	exec(`INSERT INTO file_models (id,name,path,content_type,size,parent_id,hash,created_at,updated_at,deleted_at)
		VALUES (?::uuid,'a.txt','/u1/root_folder/Docs/a.txt','text/plain',3,?::uuid,'h1',now(),now(),NULL)`, idFileA, idDocs)
	exec(`INSERT INTO file_models (id,name,path,content_type,size,parent_id,hash,created_at,updated_at,deleted_at)
		VALUES (?::uuid,'gone.txt','/u1/root_folder/Docs/gone.txt','text/plain',4,?::uuid,'h2',now(),now(),now())`, idFileGone, idDocs)
}

func colExistsPG(t *testing.T, gdb *gorm.DB, table, col string) bool {
	t.Helper()
	var n int64
	require.NoError(t, gdb.Raw(
		`SELECT count(*) FROM information_schema.columns WHERE table_name = ? AND column_name = ?`,
		table, col).Scan(&n).Error)
	return n > 0
}

func TestApply_PreservesLegacyTrashOnV141Upgrade_Postgres(t *testing.T) {
	gdb := pgGorm(t)
	seedV141PG(t, gdb)

	loader := schema.NewLoader(schema.DialectPostgres, nopLogger{}, "")
	require.NoError(t, loader.Apply(context.Background(), gdb))

	var versions []int
	require.NoError(t, gdb.Raw(`SELECT version FROM buckt_schema_migrations ORDER BY version`).Scan(&versions).Error)
	assert.Equal(t, []int{1, 2, 3}, versions)

	assert.False(t, colExistsPG(t, gdb, "file_models", "deleted_at"))
	assert.False(t, colExistsPG(t, gdb, "folder_models", "deleted_at"))
	assert.True(t, colExistsPG(t, gdb, "file_models", "user_id"))

	// nothing deleted: 3 folders + created __trash__, both files preserved
	var folderCount, fileCount int64
	require.NoError(t, gdb.Raw(`SELECT count(*) FROM folder_models`).Scan(&folderCount).Error)
	require.NoError(t, gdb.Raw(`SELECT count(*) FROM file_models`).Scan(&fileCount).Error)
	assert.Equal(t, int64(4), folderCount)
	assert.Equal(t, int64(2), fileCount)

	var trashID string
	require.NoError(t, gdb.Raw(
		`SELECT id::text FROM folder_models WHERE user_id='u1' AND name='__trash__' AND parent_id IS NULL`).Scan(&trashID).Error)
	require.NotEmpty(t, trashID)

	var oldStuffParent, goneParent string
	require.NoError(t, gdb.Raw(`SELECT parent_id::text FROM folder_models WHERE id=?::uuid`, idOldStuff).Scan(&oldStuffParent).Error)
	require.NoError(t, gdb.Raw(`SELECT parent_id::text FROM file_models WHERE id=?::uuid`, idFileGone).Scan(&goneParent).Error)
	assert.Equal(t, trashID, oldStuffParent, "trashed folder relocated to __trash__")
	assert.Equal(t, trashID, goneParent, "trashed file relocated to __trash__")

	// user_id backfilled from parent folder
	var x1User string
	require.NoError(t, gdb.Raw(`SELECT user_id FROM file_models WHERE id=?::uuid`, idFileA).Scan(&x1User).Error)
	assert.Equal(t, "u1", x1User)

	// idempotent re-run
	require.NoError(t, loader.Apply(context.Background(), gdb))
	require.NoError(t, gdb.Raw(`SELECT count(*) FROM folder_models`).Scan(&folderCount).Error)
	assert.Equal(t, int64(4), folderCount)
}

func TestApply_FreshDatabase_Postgres(t *testing.T) {
	gdb := pgGorm(t)
	loader := schema.NewLoader(schema.DialectPostgres, nopLogger{}, "")
	require.NoError(t, loader.Apply(context.Background(), gdb))

	assert.True(t, colExistsPG(t, gdb, "file_models", "user_id"))
	assert.False(t, colExistsPG(t, gdb, "file_models", "deleted_at"))

	var versions []int
	require.NoError(t, gdb.Raw(`SELECT version FROM buckt_schema_migrations ORDER BY version`).Scan(&versions).Error)
	assert.Equal(t, []int{1, 2, 3}, versions)
}
