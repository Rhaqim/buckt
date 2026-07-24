package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Rhaqim/buckt/internal/constant"
	"github.com/Rhaqim/buckt/internal/database"
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/Rhaqim/buckt/pkg/buckterr"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FolderRepository struct {
	db *database.DB
}

func NewFolderRepository(db *database.DB) domain.FolderRepository {
	return &FolderRepository{db: db}
}

// Create implements domain.FolderRepository.
func (f *FolderRepository) Create(ctx context.Context, folder *model.FolderModel) (string, error) {
	if err := f.db.DB.WithContext(ctx).Create(folder).Error; err != nil {
		return "", asAlreadyExists(err, "folder already exists")
	}
	return folder.ID.String(), nil
}

// GetFolder implements domain.FolderRepository.
// Subfolders preloaded exclude the special trash folder.
func (f *FolderRepository) GetFolder(ctx context.Context, folder_id uuid.UUID) (*model.FolderModel, error) {
	var folder model.FolderModel
	err := f.db.DB.WithContext(ctx).
		Preload("Folders", "name != ?", constant.TRASH_FOLDER_NAME).
		Preload("Files").
		Where("id = ?", folder_id).First(&folder).Error
	return &folder, asNotFound(err, "folder not found")
}

// GetRootFolder returns the user's root folder, creating it if missing.
func (f *FolderRepository) GetRootFolder(ctx context.Context, user_id string) (*model.FolderModel, error) {
	root := model.FolderModel{}

	err := f.db.DB.WithContext(ctx).Preload("Folders").Preload("Files").
		Where("name = ? AND user_id = ?", constant.ROOT_FOLDER_NAME, user_id).First(&root).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}

		path := "/" + user_id + "/" + constant.ROOT_FOLDER_NAME

		if err := f.db.DB.WithContext(ctx).Create(&model.FolderModel{
			UserID:      user_id,
			Name:        constant.ROOT_FOLDER_NAME,
			Description: "Root folder",
			Path:        path,
		}).Error; err != nil {
			return nil, err
		}

		return f.GetRootFolder(ctx, user_id)
	}

	return &root, nil
}

// GetTrashFolder returns the user's reserved top-level trash folder,
// creating it if missing. The trash folder is uniquely identified by
// (user_id, name=__trash__, parent_id IS NULL) so it cannot be confused
// with a user-created nested folder of the same name.
//
// The lookup-then-insert is wrapped in a transaction with a re-check after
// the insert attempt to handle concurrent creation races.
func (f *FolderRepository) GetTrashFolder(ctx context.Context, user_id string) (*model.FolderModel, error) {
	return lookupOrCreateTrashFolder(ctx, f.db.DB, user_id)
}

// lookupOrCreateTrashFolder is the shared implementation used by both the
// folder repository and the file repository (which can't depend on the
// folder repo directly without a circular import).
func lookupOrCreateTrashFolder(ctx context.Context, db *gorm.DB, user_id string) (*model.FolderModel, error) {
	var trash model.FolderModel

	// Fast path: try to find the existing reserved trash folder
	err := db.WithContext(ctx).
		Where("user_id = ? AND name = ? AND parent_id IS NULL", user_id, constant.TRASH_FOLDER_NAME).
		First(&trash).Error
	if err == nil {
		return &trash, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// Not found — create it inside a transaction so concurrent creators
	// don't both insert duplicate rows. After the insert (or on any error),
	// re-select to return whichever row actually won the race.
	txErr := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Re-check inside the transaction in case another writer raced ahead
		var existing model.FolderModel
		recheckErr := tx.
			Where("user_id = ? AND name = ? AND parent_id IS NULL", user_id, constant.TRASH_FOLDER_NAME).
			First(&existing).Error
		if recheckErr == nil {
			trash = existing
			return nil
		}
		if !errors.Is(recheckErr, gorm.ErrRecordNotFound) {
			return recheckErr
		}

		path := "/" + user_id + "/" + constant.TRASH_FOLDER_NAME
		newTrash := model.FolderModel{
			UserID:      user_id,
			Name:        constant.TRASH_FOLDER_NAME,
			Description: "Trash",
			Path:        path,
		}
		if err := tx.Create(&newTrash).Error; err != nil {
			// Possible race: another transaction created it just now.
			// Re-select and return that one.
			var raced model.FolderModel
			if rerr := tx.
				Where("user_id = ? AND name = ? AND parent_id IS NULL", user_id, constant.TRASH_FOLDER_NAME).
				First(&raced).Error; rerr == nil {
				trash = raced
				return nil
			}
			return err
		}
		trash = newTrash
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}
	return &trash, nil
}

// GetFolders implements domain.FolderRepository.
// Excludes the special trash folder from results.
func (f *FolderRepository) GetFolders(ctx context.Context, parent_id uuid.UUID) ([]model.FolderModel, error) {
	var folders []model.FolderModel
	err := f.db.DB.WithContext(ctx).
		Where("parent_id = ? AND name != ?", parent_id, constant.TRASH_FOLDER_NAME).
		Find(&folders).Error
	return folders, err
}

// GetFoldersPaginated implements domain.FolderRepository.
// Excludes the special trash folder from results.
func (f *FolderRepository) GetFoldersPaginated(ctx context.Context, parent_id uuid.UUID, page model.Pagination) ([]model.FolderModel, error) {
	page.Validate()
	var folders []model.FolderModel
	err := f.db.DB.WithContext(ctx).
		Where("parent_id = ? AND name != ?", parent_id, constant.TRASH_FOLDER_NAME).
		Offset(page.Offset()).
		Limit(page.PageSize).
		Order("created_at DESC").
		Find(&folders).Error
	return folders, err
}

// MoveFolder implements domain.FolderRepository.
func (f *FolderRepository) MoveFolder(ctx context.Context, folder_id uuid.UUID, new_parent_id uuid.UUID) error {
	// Reject self-move outright
	if folder_id == new_parent_id {
		return fmt.Errorf("invalid move: cannot move a folder into itself")
	}

	var newParentFolder model.FolderModel
	if err := f.db.DB.WithContext(ctx).Where("id = ?", new_parent_id).First(&newParentFolder).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("parent folder not found: %w", buckterr.ErrNotFound)
		}
		return err
	}

	var folder model.FolderModel
	if err := f.db.DB.WithContext(ctx).Where("id = ?", folder_id).First(&folder).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("folder not found: %w", buckterr.ErrNotFound)
		}
		return err
	}

	// Reject move into self by path equality (defense in depth)
	if newParentFolder.Path == folder.Path {
		return fmt.Errorf("invalid move: cannot move a folder into itself")
	}

	// Prevent moving into its own subfolder (descendant)
	if strings.HasPrefix(newParentFolder.Path, folder.Path+"/") {
		return fmt.Errorf("invalid move: cannot move a folder into its own subfolder")
	}

	oldPath := folder.Path
	newPath := strings.TrimSuffix(newParentFolder.Path, "/") + "/" + folder.Name

	if folder.Path == newPath && folder.ParentID == &newParentFolder.ID {
		return nil
	}

	return f.db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&folder).Updates(map[string]interface{}{
			"path":      newPath,
			"parent_id": newParentFolder.ID,
		}).Error; err != nil {
			return err
		}

		oldPrefix := oldPath + "/"
		newPrefix := newPath + "/"
		prefixLen := len(oldPrefix)

		if err := tx.Model(&model.FolderModel{}).
			Where("path LIKE ? ESCAPE '\\'", escapeLike(oldPrefix)+"%").
			Update("path", gorm.Expr("? || SUBSTR(path, ?)", newPrefix, prefixLen+1)).Error; err != nil {
			return err
		}

		if err := tx.Model(&model.FileModel{}).
			Where("path LIKE ? ESCAPE '\\'", escapeLike(oldPrefix)+"%").
			Update("path", gorm.Expr("? || SUBSTR(path, ?)", newPrefix, prefixLen+1)).Error; err != nil {
			return err
		}

		return nil
	})
}

// RenameFolder implements domain.FolderRepository.
func (f *FolderRepository) RenameFolder(ctx context.Context, user_id string, folder_id uuid.UUID, new_name string) error {
	var folder model.FolderModel
	if err := f.db.DB.WithContext(ctx).Where("id = ? AND user_id = ?", folder_id, user_id).First(&folder).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("folder not found: %w", buckterr.ErrNotFound)
		}
		return err
	}

	oldPath := folder.Path
	newPath := strings.TrimSuffix(folder.Path, "/"+folder.Name) + "/" + new_name

	return f.db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&folder).Updates(map[string]interface{}{
			"name": new_name,
			"path": newPath,
		}).Error; err != nil {
			return err
		}

		oldPrefix := oldPath + "/"
		newPrefix := newPath + "/"
		prefixLen := len(oldPrefix)

		if err := tx.Model(&model.FolderModel{}).
			Where("path LIKE ? ESCAPE '\\'", escapeLike(oldPrefix)+"%").
			Update("path", gorm.Expr("? || SUBSTR(path, ?)", newPrefix, prefixLen+1)).Error; err != nil {
			return err
		}

		if err := tx.Model(&model.FileModel{}).
			Where("path LIKE ? ESCAPE '\\'", escapeLike(oldPrefix)+"%").
			Update("path", gorm.Expr("? || SUBSTR(path, ?)", newPrefix, prefixLen+1)).Error; err != nil {
			return err
		}

		return nil
	})
}

// DeleteFolder moves the folder to the user's trash folder. If the folder is
// already inside the trash, it is permanently deleted instead.
//
// Returns the parent_id, the (oldPath, newPath) of the folder itself, and a
// list of (oldPath, newPath) pairs for every file under the folder. If newPath
// is empty, the folder was permanently deleted.
//
// The beforeCommit callback runs INSIDE the DB transaction after the path
// rewrites but before commit. If the callback returns an error, the entire
// transaction rolls back, leaving the DB in its original state. This makes
// the DB and backend stay consistent even if a backend operation fails midway.
func (f *FolderRepository) DeleteFolder(
	ctx context.Context,
	folder_id uuid.UUID,
	beforeCommit func(oldPath, newPath string, fileMoves []model.PathMove) error,
) (parent_id, oldPath, newPath string, fileMoves []model.PathMove, err error) {
	err = f.db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var folder model.FolderModel
		if err := tx.Preload("Folders").Preload("Files").Where("id = ?", folder_id).First(&folder).Error; err != nil {
			return err
		}

		if folder.ParentID != nil {
			parent_id = folder.ParentID.String()
		}

		// Get the user's reserved top-level trash folder (created if missing).
		// Scoped by parent_id IS NULL so it can't be confused with a user-
		// created nested folder.
		var trash model.FolderModel
		trashErr := tx.
			Where("user_id = ? AND name = ? AND parent_id IS NULL", folder.UserID, constant.TRASH_FOLDER_NAME).
			First(&trash).Error
		if errors.Is(trashErr, gorm.ErrRecordNotFound) {
			trash = model.FolderModel{
				UserID:      folder.UserID,
				Name:        constant.TRASH_FOLDER_NAME,
				Description: "Trash",
				Path:        "/" + folder.UserID + "/" + constant.TRASH_FOLDER_NAME,
			}
			if err := tx.Create(&trash).Error; err != nil {
				// Concurrent creator may have won — re-select
				var raced model.FolderModel
				if rerr := tx.
					Where("user_id = ? AND name = ? AND parent_id IS NULL", folder.UserID, constant.TRASH_FOLDER_NAME).
					First(&raced).Error; rerr == nil {
					trash = raced
				} else {
					return err
				}
			}
		} else if trashErr != nil {
			return trashErr
		}

		// Refuse to delete the trash folder itself
		if folder.ID == trash.ID {
			return fmt.Errorf("cannot delete the trash folder itself")
		}

		oldPath = folder.Path

		// If the folder is already inside the trash, permanently delete it
		if strings.HasPrefix(folder.Path, trash.Path+"/") || (folder.ParentID != nil && *folder.ParentID == trash.ID) {
			if err := tx.Delete(&folder).Error; err != nil {
				return err
			}
			newPath = ""
			if beforeCommit != nil {
				if err := beforeCommit(oldPath, "", nil); err != nil {
					return err
				}
			}
			return nil
		}

		// Move folder into trash, with a unique name to avoid collisions
		newName, err := uniqueTrashName(ctx, tx, trash.ID, folder.Name)
		if err != nil {
			return err
		}
		newPath = trash.Path + "/" + newName

		// Collect all descendant file paths BEFORE the rewrite so we can return
		// the (oldPath, newPath) pairs to the caller
		oldPrefix := oldPath + "/"
		newPrefix := newPath + "/"

		var files []model.FileModel
		if err := tx.Where("path LIKE ? ESCAPE '\\'", escapeLike(oldPrefix)+"%").Find(&files).Error; err != nil {
			return err
		}
		for _, file := range files {
			fileMoves = append(fileMoves, model.PathMove{
				Old: file.Path,
				New: newPrefix + strings.TrimPrefix(file.Path, oldPrefix),
			})
		}

		if err := tx.Model(&folder).Updates(map[string]interface{}{
			"name":      newName,
			"path":      newPath,
			"parent_id": trash.ID,
		}).Error; err != nil {
			return err
		}

		prefixLen := len(oldPrefix)

		if err := tx.Model(&model.FolderModel{}).
			Where("path LIKE ? ESCAPE '\\'", escapeLike(oldPrefix)+"%").
			Update("path", gorm.Expr("? || SUBSTR(path, ?)", newPrefix, prefixLen+1)).Error; err != nil {
			return err
		}

		if err := tx.Model(&model.FileModel{}).
			Where("path LIKE ? ESCAPE '\\'", escapeLike(oldPrefix)+"%").
			Update("path", gorm.Expr("? || SUBSTR(path, ?)", newPrefix, prefixLen+1)).Error; err != nil {
			return err
		}

		// Run backend operation BEFORE commit. If it fails, the transaction
		// rolls back and all path rewrites are undone.
		if beforeCommit != nil {
			if err := beforeCommit(oldPath, newPath, fileMoves); err != nil {
				return err
			}
		}
		return nil
	})
	return
}

// ScrubFolder permanently deletes a folder and all its contents.
func (f *FolderRepository) ScrubFolder(ctx context.Context, user_id string, folder_id uuid.UUID) (parent_id string, err error) {
	var folder model.FolderModel
	if err := f.db.DB.WithContext(ctx).Where("id = ? AND user_id = ?", folder_id, user_id).First(&folder).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", fmt.Errorf("folder not found: %w", buckterr.ErrNotFound)
		}
		return "", err
	}

	if err := f.db.DB.WithContext(ctx).Delete(&folder).Error; err != nil {
		return "", err
	}

	if folder.ParentID != nil {
		return folder.ParentID.String(), nil
	}
	return "", nil
}

// uniqueTrashName returns a folder name that doesn't collide with existing
// folders in the trash. If "Photos" already exists, it returns "Photos (2)",
// "Photos (3)", etc. Folder names are not split on extensions since folders
// don't have them.
func uniqueTrashName(ctx context.Context, db *gorm.DB, trashID uuid.UUID, name string) (string, error) {
	count, err := countFolderName(ctx, db, trashID, name)
	if err != nil {
		return "", err
	}
	if count == 0 {
		return name, nil
	}

	// Try suffixes until we find one that's free
	for i := 2; i < 10000; i++ {
		candidate := fmt.Sprintf("%s (%d)", name, i)
		count, err := countFolderName(ctx, db, trashID, candidate)
		if err != nil {
			return "", err
		}
		if count == 0 {
			return candidate, nil
		}
	}
	// Fallback: use a random uuid suffix
	return name + "-" + uuid.New().String()[:8], nil
}

// asNotFound maps GORM's record-not-found to the public buckterr.ErrNotFound
// sentinel (carrying msg as context), so callers can errors.Is against it
// without importing gorm. Any other error — including nil — passes through
// unchanged.
func asNotFound(err error, msg string) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("%s: %w", msg, buckterr.ErrNotFound)
	}
	return err
}

// asAlreadyExists maps GORM's duplicate-key error (available because the DB is
// opened with TranslateError) to buckterr.ErrAlreadyExists. Other errors pass
// through unchanged.
func asAlreadyExists(err error, msg string) error {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return fmt.Errorf("%s: %w", msg, buckterr.ErrAlreadyExists)
	}
	return err
}

// escapeLike escapes the LIKE wildcards % and _ (and the escape char itself) so
// s can be used as a literal prefix in a `LIKE ? ESCAPE '\'` clause. File and
// folder names may legitimately contain % or _; without escaping those act as
// wildcards and a prefix rewrite (move/rename/delete) would match and corrupt
// unrelated sibling paths. The escape char and clause work on both SQLite and
// Postgres.
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

func countFolderName(ctx context.Context, db *gorm.DB, parentID uuid.UUID, name string) (int64, error) {
	var count int64
	err := db.WithContext(ctx).Model(&model.FolderModel{}).
		Where("parent_id = ? AND name = ?", parentID, name).Count(&count).Error
	return count, err
}
