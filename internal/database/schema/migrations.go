package schema

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Rhaqim/buckt/internal/constant"
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Migration is a single ordered, versioned change to buckt's schema. Version
// numbers are dense and start at 1. Up runs inside a transaction together with
// the ledger insert (see Loader.runMigration), so a failed migration rolls back
// cleanly and its version is never recorded. Mirrors loom's Migration, but Up
// receives a *gorm.DB transaction so migrations can reuse AutoMigrate.
//
// Never edit or reorder an already-released migration — append a new one with
// the next version number instead.
type Migration struct {
	Version int
	Name    string
	Up      func(ctx context.Context, tx *gorm.DB, prefix string, d Dialect) error
}

// migrations is the ordered list of every schema migration. Append new
// migrations to the end with the next version number.
func migrations() []Migration {
	return []Migration{
		{
			Version: 1,
			Name:    "baseline",
			Up:      migrateBaseline,
		},
		{
			Version: 2,
			Name:    "preserve-legacy-trash",
			Up:      migratePreserveLegacyTrash,
		},
	}
}

// migrateBaseline creates or adopts the folder and file tables from the current
// GORM models. On a fresh database it creates them; on a pre-migration (v1.4.x)
// database AutoMigrate only adds what is missing (e.g. file_models.user_id),
// leaving existing tables and rows intact.
//
// AutoMigrate runs FIRST and adds file_models.user_id as NOT NULL DEFAULT ” —
// a plain ADD COLUMN, so existing rows get ” without any table rebuild. The
// backfill then updates those ” rows from each file's parent folder. Order
// matters: an earlier design backfilled a nullable column BEFORE AutoMigrate
// enforced NOT NULL, but on SQLite the enforcement triggers a table rebuild that
// discarded the freshly-backfilled values (they were not yet committed), leaving
// user_id NULL. Backfilling AFTER AutoMigrate avoids that entirely.
func migrateBaseline(ctx context.Context, tx *gorm.DB, prefix string, d Dialect) error {
	// Guard against silently orphaning data: if a non-empty prefix is
	// configured but the prefixed tables don't exist while legacy un-prefixed
	// ones do, refuse rather than create a fresh empty prefixed schema that
	// makes the operator's data look lost.
	if err := guardLegacyTables(tx, prefix); err != nil {
		return err
	}

	// tx carries the gorm NamingStrategy, so AutoMigrate creates/adopts the
	// prefixed tables automatically.
	if err := tx.AutoMigrate(&model.FolderModel{}); err != nil {
		return fmt.Errorf("folder autoMigrate: %w", err)
	}
	if err := tx.AutoMigrate(&model.FileModel{}); err != nil {
		return fmt.Errorf("file autoMigrate: %w", err)
	}
	if err := backfillFileUserIDs(tx, prefix); err != nil {
		return fmt.Errorf("file user_id backfill: %w", err)
	}
	return nil
}

// guardLegacyTables returns an actionable error when a prefix is set but the
// database still holds legacy un-prefixed buckt tables and no prefixed ones —
// the situation where blindly proceeding would create empty prefixed tables and
// strand the existing data.
func guardLegacyTables(tx *gorm.DB, prefix string) error {
	if prefix == "" {
		return nil // default schema: the un-prefixed tables ARE the real ones
	}
	legacy := tx.Migrator().HasTable("folder_models")
	prefixed := tx.Migrator().HasTable(prefix + "folder_models")
	if legacy && !prefixed {
		return fmt.Errorf(
			"table prefix %q is set but this database still has legacy un-prefixed tables "+
				"(folder_models); proceeding would create empty %sfolder_models and strand your "+
				"data. Either run with the default empty prefix, or rename your existing tables to "+
				"the %s* names before upgrading",
			prefix, prefix, prefix)
	}
	return nil
}

// backfillFileUserIDs populates <prefix>file_models.user_id from the parent
// folder for legacy rows that AutoMigrate defaulted to ” (they predate the
// column). It is a no-op on a fresh database and on databases whose files
// already carry a user_id. Orphaned files (no parent folder) are left as ” —
// the NOT NULL default AutoMigrate already applied.
func backfillFileUserIDs(tx *gorm.DB, prefix string) error {
	if !tx.Migrator().HasTable(&model.FileModel{}) {
		return nil // fresh install — nothing to backfill
	}
	if !tx.Migrator().HasColumn(&model.FileModel{}, "user_id") {
		return nil // AutoMigrate should have added it; nothing to do otherwise
	}

	files := prefix + "file_models"
	folders := prefix + "folder_models"
	return tx.Exec(fmt.Sprintf(`
		UPDATE %[1]s
		SET user_id = (
			SELECT %[2]s.user_id
			FROM %[2]s
			WHERE %[2]s.id = %[1]s.parent_id
		)
		WHERE (user_id = '' OR user_id IS NULL)
		  AND EXISTS (
			SELECT 1 FROM %[2]s WHERE %[2]s.id = %[1]s.parent_id
		  )
	`, files, folders)).Error
}

// migratePreserveLegacyTrash retires the v1.4.x soft-delete scheme WITHOUT
// destroying data. The previous cleanup code hard-deleted every row with a
// non-NULL deleted_at and dropped the column, permanently erasing trashed files
// and folders on upgrade. Instead this migration relocates each soft-deleted
// row into its owner's reserved __trash__ folder — matching how the current
// code models "trash" — and only then drops the deleted_at columns.
//
// Each relocated row keeps its existing Path unchanged. That is deliberate: the
// physical blob stays where it is, so the file remains readable without any
// backend move — correct in both flat and nested-namespace modes, and the
// migration needs no access to the storage backend. Only parent_id (and a
// de-collided name) change, so the item now lists under Trash instead of
// reappearing in its original folder.
func migratePreserveLegacyTrash(ctx context.Context, tx *gorm.DB, prefix string, d Dialect) error {
	hasFileDel := tx.Migrator().HasColumn(&model.FileModel{}, "deleted_at")
	hasFolderDel := tx.Migrator().HasColumn(&model.FolderModel{}, "deleted_at")
	if !hasFileDel && !hasFolderDel {
		return nil // fresh database or already migrated — nothing to preserve
	}

	files := prefix + "file_models"
	folders := prefix + "folder_models"
	trash := &trashResolver{tx: tx, foldersTable: folders, cache: map[string]string{}}

	// Folders first: relocating a soft-deleted folder carries its whole subtree
	// (including any soft-deleted descendants) along with it.
	if hasFolderDel {
		if err := relocateSoftDeletedFolders(tx, trash, folders); err != nil {
			return fmt.Errorf("relocate soft-deleted folders: %w", err)
		}
	}
	if hasFileDel {
		if err := relocateSoftDeletedFiles(tx, trash, files, folders); err != nil {
			return fmt.Errorf("relocate soft-deleted files: %w", err)
		}
	}

	// With no row depending on deleted_at any more, drop the columns so the
	// soft-delete scheme is fully retired. Use native ALTER TABLE DROP COLUMN
	// (SQLite 3.35+ and Postgres both support it) rather than gorm's
	// Migrator().DropColumn: on SQLite the latter rebuilds the whole table, and
	// that rebuild resets other columns (e.g. the just-backfilled user_id) to
	// their model defaults. Native DROP COLUMN leaves every other column intact.
	if hasFileDel {
		if err := tx.Exec(fmt.Sprintf(`ALTER TABLE %s DROP COLUMN deleted_at`, files)).Error; err != nil {
			return fmt.Errorf("drop %s.deleted_at: %w", files, err)
		}
	}
	if hasFolderDel {
		if err := tx.Exec(fmt.Sprintf(`ALTER TABLE %s DROP COLUMN deleted_at`, folders)).Error; err != nil {
			return fmt.Errorf("drop %s.deleted_at: %w", folders, err)
		}
	}
	return nil
}

// legacyRow is the minimal projection of a soft-deleted folder or file the
// relocation needs. Reading with explicit columns via raw SQL (rather than a
// GORM model Find) keeps this migration independent of the current model shape
// — the table still carries a deleted_at column the models no longer declare,
// and file rows may briefly hold a NULL user_id mid-migration.
type legacyRow struct {
	ID       string
	UserID   sql.NullString
	Name     string
	ParentID sql.NullString
}

func scanLegacyRows(tx *gorm.DB, query string) ([]legacyRow, error) {
	rows, err := tx.Raw(query).Rows()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []legacyRow
	for rows.Next() {
		var r legacyRow
		if err := rows.Scan(&r.ID, &r.UserID, &r.Name, &r.ParentID); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// relocateSoftDeletedFolders moves every soft-deleted folder that is the ROOT of
// a deleted subtree into its owner's trash. Deleted folders whose parent is also
// deleted are skipped — they ride along with the ancestor that gets relocated.
func relocateSoftDeletedFolders(tx *gorm.DB, trash *trashResolver, foldersTable string) error {
	deleted, err := scanLegacyRows(tx, fmt.Sprintf(
		`SELECT id, user_id, name, parent_id FROM %s WHERE deleted_at IS NOT NULL`, foldersTable))
	if err != nil {
		return err
	}
	if len(deleted) == 0 {
		return nil
	}

	deletedIDs := make(map[string]bool, len(deleted))
	for _, f := range deleted {
		deletedIDs[f.ID] = true
	}

	for _, f := range deleted {
		// Skip non-root members of a deleted subtree — the deleted ancestor
		// will move and take this folder with it.
		if f.ParentID.Valid && deletedIDs[f.ParentID.String] {
			continue
		}

		trashID, err := trash.get(f.UserID.String)
		if err != nil {
			return err
		}
		if f.ID == trashID {
			continue // never relocate the trash folder itself
		}

		name, err := uniqueName(tx, foldersTable, trashID, f.Name)
		if err != nil {
			return err
		}
		if err := tx.Exec(fmt.Sprintf(
			`UPDATE %s SET parent_id = ?, name = ? WHERE id = ?`, foldersTable),
			trashID, name, f.ID).Error; err != nil {
			return err
		}
	}
	return nil
}

// relocateSoftDeletedFiles moves every soft-deleted file whose parent folder is
// NOT itself soft-deleted into its owner's trash. Files under a deleted folder
// are left in place — they follow the folder into trash.
func relocateSoftDeletedFiles(tx *gorm.DB, trash *trashResolver, filesTable, foldersTable string) error {
	deleted, err := scanLegacyRows(tx, fmt.Sprintf(
		`SELECT id, user_id, name, parent_id FROM %s WHERE deleted_at IS NOT NULL`, filesTable))
	if err != nil {
		return err
	}
	if len(deleted) == 0 {
		return nil
	}

	softFolders, err := softDeletedFolderIDs(tx, foldersTable)
	if err != nil {
		return err
	}

	for _, f := range deleted {
		if f.ParentID.Valid && softFolders[f.ParentID.String] {
			continue // rides along with its (also-relocated) parent folder
		}

		trashID, err := trash.get(f.UserID.String)
		if err != nil {
			return err
		}

		name, err := uniqueName(tx, filesTable, trashID, f.Name)
		if err != nil {
			return err
		}
		if err := tx.Exec(fmt.Sprintf(
			`UPDATE %s SET parent_id = ?, name = ? WHERE id = ?`, filesTable),
			trashID, name, f.ID).Error; err != nil {
			return err
		}
	}
	return nil
}

// softDeletedFolderIDs returns the set of folder ids with a non-NULL deleted_at.
func softDeletedFolderIDs(tx *gorm.DB, foldersTable string) (map[string]bool, error) {
	rows, err := tx.Raw(fmt.Sprintf(`SELECT id FROM %s WHERE deleted_at IS NOT NULL`, foldersTable)).Rows()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}

// trashResolver looks up (creating if needed) each owner's reserved top-level
// __trash__ folder, caching the id so a run with many soft-deleted rows issues
// one lookup per owner rather than one per row.
type trashResolver struct {
	tx           *gorm.DB
	foldersTable string
	cache        map[string]string
}

func (t *trashResolver) get(userID string) (string, error) {
	if id, ok := t.cache[userID]; ok {
		return id, nil
	}

	var id string
	if err := t.tx.Raw(fmt.Sprintf(
		`SELECT id FROM %s WHERE user_id = ? AND name = ? AND parent_id IS NULL LIMIT 1`, t.foldersTable),
		userID, constant.TRASH_FOLDER_NAME,
	).Scan(&id).Error; err != nil {
		return "", err
	}
	if id != "" {
		t.cache[userID] = id
		return id, nil
	}

	// Create the reserved trash folder with explicit columns so we depend only
	// on the post-baseline shape (id, user_id, parent_id, name, description,
	// path, created_at, updated_at). deleted_at, still present here, defaults
	// NULL and is dropped once relocation completes.
	newID := uuid.New().String()
	now := time.Now()
	path := "/" + userID + "/" + constant.TRASH_FOLDER_NAME
	if err := t.tx.Exec(fmt.Sprintf(
		`INSERT INTO %s (id, user_id, parent_id, name, description, path, created_at, updated_at)
		 VALUES (?, ?, NULL, ?, ?, ?, ?, ?)`, t.foldersTable),
		newID, userID, constant.TRASH_FOLDER_NAME, "Trash", path, now, now,
	).Error; err != nil {
		return "", err
	}
	t.cache[userID] = newID
	return newID, nil
}

// uniqueName returns a name that does not collide with an existing row under
// parentID in table, suffixing " (2)", " (3)", ... as needed. table is an
// engine-controlled constant ("folder_models"/"file_models"), never user input.
func uniqueName(tx *gorm.DB, table, parentID, name string) (string, error) {
	count := func(candidate string) (int64, error) {
		var n int64
		err := tx.Raw(
			fmt.Sprintf(`SELECT count(*) FROM %s WHERE parent_id = ? AND name = ?`, table),
			parentID, candidate,
		).Scan(&n).Error
		return n, err
	}

	n, err := count(name)
	if err != nil {
		return "", err
	}
	if n == 0 {
		return name, nil
	}
	for i := 2; i < 10000; i++ {
		candidate := fmt.Sprintf("%s (%d)", name, i)
		n, err := count(candidate)
		if err != nil {
			return "", err
		}
		if n == 0 {
			return candidate, nil
		}
	}
	// Extremely unlikely fallback: a random suffix guaranteed not to collide.
	return name + "-" + uuid.New().String()[:8], nil
}
