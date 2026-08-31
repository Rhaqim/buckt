package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Rhaqim/buckt/internal/database"
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/Rhaqim/buckt/pkg/buckterr"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FileRepository struct {
	db *database.DB
}

func NewFileRepository(db *database.DB) domain.FileRepository {
	return &FileRepository{db: db}
}

// Create implements domain.FileRepository.
func (f *FileRepository) Create(ctx context.Context, file *model.FileModel) error {
	return asAlreadyExists(f.db.DB.WithContext(ctx).Create(file).Error, "file already exists")
}

// GetFile implements domain.FileRepository.
func (f *FileRepository) GetFile(ctx context.Context, id uuid.UUID) (*model.FileModel, error) {
	var file model.FileModel
	err := f.db.DB.WithContext(ctx).First(&file, id).Error
	return &file, asNotFound(err, "file not found")
}

// GetFiles implements domain.FileRepository.
func (f *FileRepository) GetFiles(ctx context.Context, parent_id uuid.UUID) ([]*model.FileModel, error) {
	var files []*model.FileModel
	err := f.db.DB.WithContext(ctx).Where("parent_id = ?", parent_id).Find(&files).Error
	return files, err
}

// GetFilesPaginated implements domain.FileRepository.
func (f *FileRepository) GetFilesPaginated(ctx context.Context, parent_id uuid.UUID, page model.Pagination) ([]*model.FileModel, error) {
	page.Validate()
	var files []*model.FileModel
	err := f.db.DB.WithContext(ctx).
		Where("parent_id = ?", parent_id).
		Offset(page.Offset()).
		Limit(page.PageSize).
		Order("created_at DESC").
		Find(&files).Error
	return files, err
}

// MoveFile implements domain.FileRepository.
func (f *FileRepository) MoveFile(ctx context.Context, file_id uuid.UUID, new_parent_id uuid.UUID) (string, string, error) {
	var newParentFolder model.FolderModel

	if err := f.db.DB.WithContext(ctx).Where("id = ?", new_parent_id).First(&newParentFolder).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", "", fmt.Errorf("parent folder not found: %w", buckterr.ErrNotFound)
		}
		return "", "", err
	}

	var file model.FileModel
	if err := f.db.DB.WithContext(ctx).First(&file, file_id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", "", fmt.Errorf("file not found: %w", buckterr.ErrNotFound)
		}
		return "", "", err
	}

	oldPath := file.Path

	file.ParentID = new_parent_id
	file.Path = newParentFolder.Path + "/" + file.Name

	if err := f.db.DB.WithContext(ctx).Save(&file).Error; err != nil {
		return "", "", err
	}

	return oldPath, file.Path, nil
}

// RenameFile implements domain.FileRepository.
func (f *FileRepository) RenameFile(ctx context.Context, file_id uuid.UUID, new_name string) error {
	var file model.FileModel
	if err := f.db.DB.WithContext(ctx).First(&file, file_id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("file not found: %w", buckterr.ErrNotFound)
		}
		return err
	}

	file.Name = new_name

	return f.db.DB.WithContext(ctx).Save(&file).Error
}

// Update implements domain.FileRepository.
func (f *FileRepository) Update(ctx context.Context, file *model.FileModel) error {
	return f.db.DB.WithContext(ctx).Save(file).Error
}

// DeleteFile moves the file to the user's trash folder. If the file is already
// in trash, it is permanently deleted instead.
//
// Returns the file's old and new paths so the caller can move the underlying
// data on the storage backend. If newPath is empty, the file was permanently
// deleted (already in trash). If newPath equals oldPath, the file was moved
// in flat-namespace mode and no backend op is needed.
//
// The beforeCommit callback runs INSIDE the DB transaction after the path
// rewrite but before commit. If the callback returns an error, the entire
// transaction rolls back, leaving the DB in its original state. This makes
// the DB and backend stay consistent even if the backend operation fails.
func (f *FileRepository) DeleteFile(
	ctx context.Context,
	id uuid.UUID,
	beforeCommit func(oldPath, newPath string) error,
) (oldPath, newPath string, err error) {
	err = f.db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var file model.FileModel
		if err := tx.First(&file, id).Error; err != nil {
			return err
		}

		oldPath = file.Path

		// Defense in depth: if user_id is empty (legacy data not yet backfilled),
		// derive it from the parent folder so we don't move files into a phantom
		// "empty user" trash bin.
		if file.UserID == "" {
			var parent model.FolderModel
			if err := tx.Select("user_id").First(&parent, file.ParentID).Error; err == nil && parent.UserID != "" {
				file.UserID = parent.UserID
				_ = tx.Model(&file).Update("user_id", parent.UserID).Error
			}
		}

		// If we still don't have a user_id, refuse to delete.
		if file.UserID == "" {
			return fmt.Errorf("cannot delete file %s: unable to resolve owner user_id (legacy row with missing user_id and no parent folder user_id)", id)
		}

		// Lookup or create the user's trash folder
		trash, err := getOrCreateTrashFolder(ctx, tx, file.UserID)
		if err != nil {
			return err
		}

		// Detect whether the file is already in trash. There are three cases
		// to cover so that the smart re-delete behavior works for nested files:
		//   1. The file's direct parent IS the trash folder
		//   2. Nested mode: the file's path lives under the trash prefix
		//      (e.g., file in a folder that was moved to trash)
		//   3. Flat-namespace mode: the file's parent folder is anywhere
		//      under trash (path-prefix check on the parent folder)
		inTrash := file.ParentID == trash.ID
		if !inTrash && strings.HasPrefix(file.Path, trash.Path+"/") {
			inTrash = true
		}
		if !inTrash {
			var parentFolder model.FolderModel
			if perr := tx.Select("path").First(&parentFolder, file.ParentID).Error; perr == nil {
				if parentFolder.Path == trash.Path || strings.HasPrefix(parentFolder.Path, trash.Path+"/") {
					inTrash = true
				}
			}
		}
		if inTrash {
			if err := tx.Delete(&model.FileModel{}, id).Error; err != nil {
				return err
			}
			// Run backend cleanup BEFORE commit — failure rolls back the row deletion
			if beforeCommit != nil {
				if err := beforeCommit(oldPath, ""); err != nil {
					return err
				}
			}
			return nil
		}

		// Move file into trash with a unique name
		newName, err := uniqueFileTrashName(ctx, tx, trash.ID, file.Name)
		if err != nil {
			return err
		}

		// Detect flat-namespace mode by checking for any directory separator
		// (either / or \, since filepath.Join on Windows can produce backslashes).
		isFlatPath := !strings.ContainsAny(file.Path, "/\\")

		updates := map[string]any{
			"name":      newName,
			"parent_id": trash.ID,
			// Remember where it came from so restore can return it here.
			"origin_parent_id": file.ParentID,
		}
		if !isFlatPath {
			newPath = trash.Path + "/" + newName
			updates["path"] = newPath
		} else {
			// Path stays the same in flat mode; signal "no backend move needed"
			// by returning the same value for old and new.
			newPath = file.Path
		}

		if err := tx.Model(&file).Updates(updates).Error; err != nil {
			return err
		}

		// Run backend operation BEFORE commit. If it fails, the transaction
		// rolls back and the path rewrite is undone.
		if beforeCommit != nil {
			if err := beforeCommit(oldPath, newPath); err != nil {
				return err
			}
		}
		return nil
	})
	return
}

// RestoreFile moves a trashed file to target (its origin folder, resolved by
// the caller, or root as a fallback) and clears its origin marker. Mirrors
// DeleteFile: the beforeCommit callback runs inside the transaction after the
// path rewrite, so a backend-move failure rolls the whole thing back. Returns
// the old and new paths (newPath == oldPath in flat-namespace mode = no backend
// op needed).
func (f *FileRepository) RestoreFile(
	ctx context.Context,
	id uuid.UUID,
	target uuid.UUID,
	beforeCommit func(oldPath, newPath string) error,
) (oldPath, newPath string, err error) {
	err = f.db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var file model.FileModel
		if err := tx.First(&file, id).Error; err != nil {
			return err
		}

		var dest model.FolderModel
		if err := tx.First(&dest, target).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("restore target folder not found: %w", buckterr.ErrNotFound)
			}
			return err
		}

		oldPath = file.Path

		// Dedupe the name at the destination (uniqueFileTrashName is generic —
		// it just finds a non-colliding name under a parent), in case a new file
		// with the same name now lives there.
		newName, err := uniqueFileTrashName(ctx, tx, dest.ID, file.Name)
		if err != nil {
			return err
		}

		isFlatPath := !strings.ContainsAny(file.Path, "/\\")

		updates := map[string]any{
			"name":      newName,
			"parent_id": dest.ID,
			// Clear the origin marker — the file is no longer in trash. A map
			// update writes NULL for a nil value (struct updates would skip it).
			"origin_parent_id": nil,
		}
		if !isFlatPath {
			newPath = dest.Path + "/" + newName
			updates["path"] = newPath
		} else {
			newPath = file.Path
		}

		if err := tx.Model(&file).Updates(updates).Error; err != nil {
			return err
		}

		if beforeCommit != nil {
			if err := beforeCommit(oldPath, newPath); err != nil {
				return err
			}
		}
		return nil
	})
	return
}

// FindByHash returns a file with the given content hash under the same owner
// and parent folder, or ErrNotFound. Scoped to the parent so a match is always
// a live file in the upload target (never a trashed one, which lives under the
// __trash__ folder).
func (f *FileRepository) FindByHash(ctx context.Context, user_id string, parent_id uuid.UUID, hash string) (*model.FileModel, error) {
	var file model.FileModel
	err := f.db.DB.WithContext(ctx).
		Where("user_id = ? AND parent_id = ? AND hash = ?", user_id, parent_id, hash).
		First(&file).Error
	if err != nil {
		return nil, asNotFound(err, "no duplicate file")
	}
	return &file, nil
}

// SetMetadata replaces the file's arbitrary key/value metadata. The map is
// stored as JSON via the model's field serializer. We load the row, then use
// Select(...).Updates(struct) — a field-aware update — so GORM applies the
// serializer (a bare Update("metadata", map) would try to bind the map straight
// to the driver and fail). Select forces the column to be written even when the
// map is nil, so passing nil clears the metadata.
func (f *FileRepository) SetMetadata(ctx context.Context, id uuid.UUID, metadata map[string]string) error {
	var file model.FileModel
	if err := f.db.DB.WithContext(ctx).First(&file, id).Error; err != nil {
		return asNotFound(err, "file not found")
	}
	file.Metadata = metadata
	return f.db.DB.WithContext(ctx).Model(&file).Select("Metadata").Updates(file).Error
}

// ScrubFile permanently deletes a file regardless of its location.
func (f *FileRepository) ScrubFile(ctx context.Context, id uuid.UUID) error {
	return f.db.DB.WithContext(ctx).Delete(&model.FileModel{}, id).Error
}

// SetExpiry sets (at != nil) or clears (at == nil) the file's expiry. Like
// SetMetadata it loads the row then does a field-aware Select(...).Updates so
// the column is written even when the value is nil (clearing the expiry).
func (f *FileRepository) SetExpiry(ctx context.Context, id uuid.UUID, at *time.Time) error {
	var file model.FileModel
	if err := f.db.DB.WithContext(ctx).First(&file, id).Error; err != nil {
		return asNotFound(err, "file not found")
	}
	file.ExpiresAt = at
	return f.db.DB.WithContext(ctx).Model(&file).Select("ExpiresAt").Updates(file).Error
}

// FindExpired returns up to limit files whose expires_at is set and at or before
// now, ordered oldest-expiry first so the most-overdue files are purged first.
func (f *FileRepository) FindExpired(ctx context.Context, now time.Time, limit int) ([]*model.FileModel, error) {
	var files []*model.FileModel
	q := f.db.DB.WithContext(ctx).
		Where("expires_at IS NOT NULL AND expires_at <= ?", now).
		Order("expires_at ASC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Find(&files).Error; err != nil {
		return nil, err
	}
	return files, nil
}

// getOrCreateTrashFolder returns the user's trash folder, creating it if missing.
// This duplicates the FolderRepository logic to avoid a cross-repo dependency.
func getOrCreateTrashFolder(ctx context.Context, db *gorm.DB, user_id string) (*model.FolderModel, error) {
	// Delegate to the shared helper in folder.go which enforces the
	// reserved-top-level invariant (parent_id IS NULL) and handles
	// concurrent-create races via a transaction + re-select.
	return lookupOrCreateTrashFolder(ctx, db, user_id)
}

// uniqueFileTrashName returns a file name that doesn't collide with existing
// files in the trash folder. If "photo.png" already exists, it returns
// "photo (2).png", "photo (3).png", etc., preserving the extension.
func uniqueFileTrashName(ctx context.Context, db *gorm.DB, trashID uuid.UUID, name string) (string, error) {
	return uniqueChildName(name, true, func(candidate string) (int64, error) {
		return countFileName(ctx, db, trashID, candidate)
	})
}

func countFileName(ctx context.Context, db *gorm.DB, parentID uuid.UUID, name string) (int64, error) {
	var count int64
	err := db.WithContext(ctx).Model(&model.FileModel{}).
		Where("parent_id = ? AND name = ?", parentID, name).Count(&count).Error
	return count, err
}
