package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/Rhaqim/buckt/internal/utils"
	"github.com/Rhaqim/buckt/pkg/buckterr"
	"github.com/google/uuid"
)

type FileService struct {
	flatNameSpaces   bool
	maxFileSize      int64
	backendOpTimeout time.Duration

	logger domain.BucktLogger

	cache domain.CacheManager
	repo  domain.FileRepository

	folderService domain.FolderService
	fileBackend   domain.FileBackend
}

func NewFileService(
	bucktLogger domain.BucktLogger,

	cache domain.CacheManager,
	fileRepository domain.FileRepository,

	folderService domain.FolderService,
	fileBackend domain.FileBackend,

	flatNameSpaces bool,
	maxFileSize int64,
	backendOpTimeout time.Duration,
) domain.FileService {
	bucktLogger.Info("🚀 Initialising file services")
	return &FileService{
		logger: bucktLogger,

		cache: cache,

		repo: fileRepository,

		folderService: folderService,
		fileBackend:   fileBackend,

		flatNameSpaces:   flatNameSpaces,
		maxFileSize:      maxFileSize,
		backendOpTimeout: backendOpTimeout,
	}
}

// CreateFile implements domain.FileService.
func (f *FileService) CreateFile(ctx context.Context, user_id, parent_id, file_name, content_type string, file_data []byte) (string, error) {
	var err error

	// Validate the file name before it is ever joined to a folder path or used
	// as a backend object key. Without this a name like "../../<victim>/x" would
	// escape the parent folder's subtree (local backend) or be written verbatim
	// as a cross-tenant object key (cloud backends).
	if err := utils.ValidateFileName(file_name); err != nil {
		return "", fmt.Errorf("invalid file name: %w", buckterr.ErrInvalidName)
	}

	// Enforce max file size
	if f.maxFileSize > 0 && int64(len(file_data)) > f.maxFileSize {
		return "", fmt.Errorf("file size %d exceeds maximum allowed size %d bytes: %w", len(file_data), f.maxFileSize, buckterr.ErrFileTooLarge)
	}

	// Detect and validate content type from actual file bytes
	detectedType := http.DetectContentType(file_data)
	if content_type == "" || content_type == "application/octet-stream" {
		content_type = detectedType
	}

	// Get the parent folder
	parentFolder, err := f.folderService.GetFolder(ctx, user_id, parent_id)
	if err != nil {
		parentFolder, err = f.folderService.GetRootFolder(ctx, user_id)
		if err != nil {
			return "", err
		}
	}

	// Get the file path
	path := filepath.Join(parentFolder.Path, file_name)

	// if flat namespaces is enabled save files in root with uuid as name
	if f.flatNameSpaces {
		ext := filepath.Ext(file_name)
		path = uuid.New().String() + ext
	}

	// Calculate the file hash from content only (not path)
	h := sha256.New()
	h.Write(file_data)
	hash := fmt.Sprintf("%x", h.Sum(nil))

	// Size of the file
	fileSize := int64(len(file_data))

	// Create the file model
	file := &model.FileModel{
		UserID:      user_id,
		ParentID:    parentFolder.ID,
		Name:        file_name,
		Path:        path,
		Hash:        hash,
		ContentType: content_type,
		Size:        fileSize,
	}

	// Create the file
	if err = f.repo.Create(ctx, file); err != nil {
		return "", f.logger.WrapError("failed to create file", err)
	}

	// Write the file to the backend. If this fails, the metadata row was already
	// committed above, so we compensate by removing it — otherwise the DB would
	// hold a FileModel pointing at a blob that doesn't exist. Best-effort: if the
	// rollback itself fails we log it (the caller needs the original Put error).
	//
	// We deliberately keep this order — metadata first, blob second — rather than
	// writing the blob first: in nested-namespace mode the blob path is
	// parentFolder.Path/name, so a blob-first write could overwrite an existing
	// file's content before repo.Create's unique-name check runs, then delete it
	// on rollback, destroying an unrelated file.
	if err := f.fileBackend.Put(ctx, file.Path, file_data); err != nil {
		if scrubErr := f.repo.ScrubFile(ctx, file.ID); scrubErr != nil {
			f.logger.Warn("failed to roll back file metadata after backend write failure: " + scrubErr.Error())
		}
		return "", fmt.Errorf("failed to write file to backend: %w", err)
	}

	return file.ID.String(), nil
}

// GetFile implements domain.FileService.
// Subtle: this method shadows the method (FileRepository).GetFile of FileService.repo.
func (f *FileService) GetFile(ctx context.Context, file_id string) (*model.FileModel, error) {
	fileID, err := uuid.Parse(file_id)
	if err != nil {
		return nil, fmt.Errorf("invalid id: %w", buckterr.ErrInvalidID)
	}

	var file *model.FileModel

	// Check cache first
	if f.cache != nil {
		cached, err := f.cache.GetBucktValue(ctx, file_id)
		if err != nil {
			f.logger.Warn("failed to get file metadata from cache: " + err.Error())
		}

		if cached != nil {
			cachedStr, ok := cached.(string)
			if ok {
				var cachedFile model.FileModel
				if jsonErr := json.Unmarshal([]byte(cachedStr), &cachedFile); jsonErr == nil {
					file = &cachedFile
				}
			}
		}
	}

	// If not found in cache, fetch from repository
	if file == nil {
		file, err = f.repo.GetFile(ctx, fileID)
		if err != nil {
			return nil, f.logger.WrapError("failed to get file metadata", err)
		}

		// Store metadata in cache (without file data)
		if f.cache != nil {
			jsonData, _ := json.Marshal(file) // Ignore errors for now
			_ = f.cache.SetBucktValue(ctx, file_id, string(jsonData))
		}
	}

	data, err := f.fileBackend.Get(ctx, file.Path)
	if err != nil {
		return nil, f.logger.WrapError("failed to get file data", err)
	}

	file.Data = data

	return file, nil
}

// GetFileStream implements domain.FileService.
// Subtle: this method shadows the method (FileBackend).GetFilStream of FileService.fileBackend.
func (f *FileService) GetFileStream(ctx context.Context, file_id string) (*model.FileModel, io.ReadCloser, error) {
	fileID, err := uuid.Parse(file_id)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid id: %w", buckterr.ErrInvalidID)
	}

	var file *model.FileModel

	// Check cache first
	if f.cache != nil {
		cached, err := f.cache.GetBucktValue(ctx, file_id)
		if err == nil && cached != nil { // Ensure cached value is not nil
			cachedStr, ok := cached.(string)
			if ok { // Ensure type assertion succeeds
				var cachedFile model.FileModel
				if jsonErr := json.Unmarshal([]byte(cachedStr), &cachedFile); jsonErr == nil {
					file = &cachedFile
				}
			}
		}
	}

	// If not found in cache, fetch from repository
	if file == nil {
		file, err = f.repo.GetFile(ctx, fileID)
		if err != nil {
			return nil, nil, f.logger.WrapError("failed to get file metadata", err)
		}

		// Store metadata in cache (without file data)
		if f.cache != nil {
			jsonData, _ := json.Marshal(file) // Ignore errors for now
			_ = f.cache.SetBucktValue(ctx, file_id, string(jsonData))
		}
	}

	// Fetch actual file data separately
	fileStream, err := f.fileBackend.Stream(ctx, file.Path)
	if err != nil {
		return nil, nil, f.logger.WrapError("failed to get file data", err)
	}

	return file, fileStream, nil
}

func (f *FileService) getFiles(ctx context.Context, parent_id string) ([]*model.FileModel, error) {
	parentID, err := uuid.Parse(parent_id)
	if err != nil {
		return nil, fmt.Errorf("invalid id: %w", buckterr.ErrInvalidID)
	}

	var files []*model.FileModel

	// Generate cache key
	cacheKey := fmt.Sprintf("files:%s", parent_id)

	// Check cache first
	if f.cache != nil {
		cached, err := f.cache.GetBucktValue(ctx, cacheKey)
		if err == nil && cached != nil {
			if cachedStr, ok := cached.(string); ok {
				var cachedFiles []*model.FileModel
				if jsonErr := json.Unmarshal([]byte(cachedStr), &cachedFiles); jsonErr == nil {
					files = cachedFiles
				}
			}
		}
	}

	// If not found in cache, fetch from repository
	if len(files) == 0 {
		files, err = f.repo.GetFiles(ctx, parentID)
		if err != nil {
			return nil, f.logger.WrapError("failed to get files", err)
		}

		// Store metadata in cache (without file data)
		if f.cache != nil {
			jsonData, _ := json.Marshal(files) // Ignore errors for now
			_ = f.cache.SetBucktValue(ctx, cacheKey, string(jsonData))
		}
	}

	return files, nil
}

func (f *FileService) GetFilesMetadata(ctx context.Context, parent_id string) ([]model.FileModel, error) {
	files, err := f.getFiles(ctx, parent_id)
	if err != nil {
		return nil, err
	}

	// Convert []*model.FileModel to []model.FileModel
	var fileModels []model.FileModel
	for _, file := range files {
		fileModels = append(fileModels, *file)
	}

	return fileModels, nil
}

// GetFiles implements domain.FileService.
// Subtle: this method shadows the method (FileRepository).GetFiles of FileService.repo.
func (f *FileService) GetFiles(ctx context.Context, parent_id string) ([]model.FileModel, error) {
	files, err := f.getFiles(ctx, parent_id)
	if err != nil {
		return nil, err
	}

	// Fetch actual file data separately
	var fileModels []model.FileModel
	for _, file := range files {
		fileData, err := f.fileBackend.Get(ctx, file.Path)
		if err != nil {
			return nil, f.logger.WrapError("failed to get file data", err)
		}
		file.Data = fileData
		fileModels = append(fileModels, *file)
	}

	return fileModels, nil
}

// MoveFile implements domain.FileService.
func (f *FileService) MoveFile(ctx context.Context, file_id string, new_parent_id string) error {
	fileID, err := uuid.Parse(file_id)
	if err != nil {
		return fmt.Errorf("invalid id: %w", buckterr.ErrInvalidID)
	}

	newParentID, err := uuid.Parse(new_parent_id)
	if err != nil {
		return fmt.Errorf("invalid id: %w", buckterr.ErrInvalidID)
	}

	// Move the file
	oldPath, newPath, err := f.repo.MoveFile(ctx, fileID, newParentID)
	if err != nil {
		return f.logger.WrapError("failed to move file", err)
	}

	if !f.flatNameSpaces {
		// Move the file in the file system
		if err := f.fileBackend.Move(ctx, oldPath, newPath); err != nil {
			return f.logger.WrapError("failed to move file", err)
		}
	}

	return nil
}

// RestoreFile moves a trashed file back to its original location (or root if
// that location no longer exists) and clears the trash origin marker. The
// backend blob is physically moved in nested-namespace mode.
func (f *FileService) RestoreFile(ctx context.Context, user_id, file_id string) error {
	fileID, err := uuid.Parse(file_id)
	if err != nil {
		return fmt.Errorf("invalid id: %w", buckterr.ErrInvalidID)
	}

	file, err := f.repo.GetFile(ctx, fileID)
	if err != nil {
		return f.logger.WrapError("failed to get file", err)
	}

	target, err := f.resolveRestoreTarget(ctx, user_id, file.OriginParentID)
	if err != nil {
		return err
	}

	// Parent and path change — drop any cached copy.
	if f.cache != nil {
		_ = f.cache.DeleteBucktValue(ctx, file_id)
	}

	_, _, err = f.repo.RestoreFile(ctx, fileID, target, func(oldPath, newPath string) error {
		if f.flatNameSpaces || oldPath == newPath {
			return nil
		}
		opCtx, cancel := f.backendCtx(ctx)
		defer cancel()
		return f.fileBackend.Move(opCtx, oldPath, newPath)
	})
	if err != nil {
		return f.logger.WrapError("failed to restore file", err)
	}

	return nil
}

// resolveRestoreTarget picks where a trashed item should be restored to: its
// recorded origin folder if that folder still exists outside the trash,
// otherwise the user's root folder. Returns the root for a nil origin (legacy
// trashed items predating origin tracking).
func (f *FileService) resolveRestoreTarget(ctx context.Context, user_id string, origin *uuid.UUID) (uuid.UUID, error) {
	root, err := f.folderService.GetRootFolder(ctx, user_id)
	if err != nil {
		return uuid.Nil, f.logger.WrapError("failed to resolve root folder", err)
	}
	if origin == nil {
		return root.ID, nil
	}

	// GetFolder falls back to root when the folder is gone, so a root result
	// means the origin no longer exists — restore to root.
	originFolder, err := f.folderService.GetFolder(ctx, user_id, origin.String())
	if err != nil || originFolder == nil || originFolder.ID == root.ID {
		return root.ID, nil
	}

	// If the origin folder was itself moved to trash, don't restore back into
	// the trash — fall back to root.
	if trash, terr := f.folderService.GetTrashFolder(ctx, user_id); terr == nil && trash != nil {
		if originFolder.ID == trash.ID || originFolder.Path == trash.Path || strings.HasPrefix(originFolder.Path, trash.Path+"/") {
			return root.ID, nil
		}
	}

	return originFolder.ID, nil
}

// RenameFile implements domain.FileService.
func (f *FileService) RenameFile(ctx context.Context, file_id string, new_name string) error {
	if err := utils.ValidateFileName(new_name); err != nil {
		return fmt.Errorf("invalid file name: %w", buckterr.ErrInvalidName)
	}

	fileID, err := uuid.Parse(file_id)
	if err != nil {
		return fmt.Errorf("invalid id: %w", buckterr.ErrInvalidID)
	}

	// Rename the file
	if err := f.repo.RenameFile(ctx, fileID, new_name); err != nil {
		return f.logger.WrapError("failed to rename file", err)
	}

	return nil
}

// UpdateFile implements domain.FileService.
func (f *FileService) UpdateFile(ctx context.Context, user_id, file_id string, new_file_name string, new_file_data []byte) error {
	if err := utils.ValidateFileName(new_file_name); err != nil {
		return fmt.Errorf("invalid file name: %w", buckterr.ErrInvalidName)
	}

	fileID, err := uuid.Parse(file_id)
	if err != nil {
		return fmt.Errorf("invalid id: %w", buckterr.ErrInvalidID)
	}

	file, err := f.repo.GetFile(ctx, fileID)
	if err != nil {
		return f.logger.WrapError("failed to get file", err)
	}

	// Get the parent folder
	parentFolder, err := f.folderService.GetFolder(ctx, user_id, file.ParentID.String())
	if err != nil {
		return err
	}

	// Get the new file path. Use filepath.Join (not raw concatenation) so the
	// name is cleaned; combined with validateFileName above this keeps the path
	// confined to the parent folder.
	newPath := filepath.Join(parentFolder.Path, new_file_name)

	// Calculate the new file hash, for data verification
	newHash := fmt.Sprintf("%x", sha256.Sum256(new_file_data))

	// Update the file model
	file.Name = new_file_name
	file.Path = newPath
	file.Hash = newHash

	// Update the file in the file system
	if err := f.fileBackend.Put(ctx, newPath, new_file_data); err != nil {
		return err
	}

	// Update the file
	if err := f.repo.Update(ctx, file); err != nil {
		return f.logger.WrapError("failed to update file", err)
	}

	return nil
}

// DeleteFile implements domain.FileService.
// Subtle: this method shadows the method (FileRepository).DeleteFile of FileService.repo.
func (f *FileService) DeleteFile(ctx context.Context, file_id string) (string, error) {
	var parentID string

	fileID, err := uuid.Parse(file_id)
	if err != nil {
		return parentID, fmt.Errorf("invalid id: %w", buckterr.ErrInvalidID)
	}

	var file *model.FileModel

	// Check cache first
	if f.cache != nil {
		cached, err := f.cache.GetBucktValue(ctx, file_id)
		if err == nil {
			var cachedFile model.FileModel
			if jsonErr := json.Unmarshal([]byte(cached.(string)), &cachedFile); jsonErr == nil {
				file = &cachedFile
			}

			// Delete from cache
			_ = f.cache.DeleteBucktValue(ctx, file_id)
		}
	}

	// If not found in cache, fetch from repository
	if file == nil {
		file, err = f.repo.GetFile(ctx, fileID)
		if err != nil {
			return parentID, f.logger.WrapError("failed to get file metadata", err)
		}
	}

	// Delete the file with backend coordination. The backend op runs INSIDE
	// the DB transaction (so a backend failure rolls back the path rewrite),
	// but is bounded by backendOpTimeout to cap how long the transaction can
	// stay open while waiting on storage I/O.
	_, _, err = f.repo.DeleteFile(ctx, fileID, func(oldPath, newPath string) error {
		opCtx, cancel := f.backendCtx(ctx)
		defer cancel()
		switch {
		case newPath == "":
			// Permanent deletion (was already in trash). Remove the blob from
			// the backend. Works in both flat and nested modes because oldPath
			// always reflects the actual stored location.
			return f.fileBackend.Delete(opCtx, oldPath)
		case oldPath == newPath:
			// Flat mode — only parent_id changed, blob stays put.
			return nil
		default:
			// Nested mode — physically move the blob to its new path.
			return f.fileBackend.Move(opCtx, oldPath, newPath)
		}
	})
	if err != nil {
		return parentID, f.logger.WrapError("failed to delete file", err)
	}

	return file.ParentID.String(), nil
}

func (f *FileService) ScrubFile(ctx context.Context, file_id string) (string, error) {
	var parentID string

	fileID, err := uuid.Parse(file_id)
	if err != nil {
		return parentID, fmt.Errorf("invalid id: %w", buckterr.ErrInvalidID)
	}

	var file *model.FileModel

	// Check cache first
	if f.cache != nil {
		cached, err := f.cache.GetBucktValue(ctx, file_id)
		if err == nil {
			var cachedFile model.FileModel
			if jsonErr := json.Unmarshal([]byte(cached.(string)), &cachedFile); jsonErr == nil {
				file = &cachedFile
			}

			// Delete from cache
			_ = f.cache.DeleteBucktValue(ctx, file_id)
		}
	}

	// If not found in cache, fetch from repository
	if file == nil {
		file, err = f.repo.GetFile(ctx, fileID)
		if err != nil {
			return parentID, f.logger.WrapError("failed to get file metadata", err)
		}
	}

	// Delete the file from the file system
	if err := f.fileBackend.Delete(ctx, file.Path); err != nil {
		return parentID, err
	}

	// Delete the file
	if err := f.repo.ScrubFile(ctx, fileID); err != nil {
		return parentID, f.logger.WrapError("failed to delete file", err)
	}

	return file.ParentID.String(), nil
}

// backendCtx returns a context with backendOpTimeout applied (if positive).
// Returns the original context unchanged if no timeout is configured.
func (f *FileService) backendCtx(parent context.Context) (context.Context, context.CancelFunc) {
	if f.backendOpTimeout <= 0 {
		return parent, func() {}
	}
	return context.WithTimeout(parent, f.backendOpTimeout)
}
