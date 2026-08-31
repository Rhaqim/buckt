// Description: Buckt is a simple file storage service that allows users to upload, download, and manage files and folders.
// It provides a simple API for managing files and folders, as well as a web interface for interacting with the service.
// Buckt supports multiple storage backends, including local storage and cloud storage providers.
// The service can be configured to use a specific storage backend, or it can be used as a standalone service with local storage.
// Buckt is built using Go and provides a simple and easy-to-use API for managing files and folders.
// It is designed to be lightweight and easy to deploy, making it ideal for small projects and personal use.
// The service is extensible and can be customized to support additional features and functionality.

package buckt

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"github.com/Rhaqim/buckt/internal/backend"
	"github.com/Rhaqim/buckt/internal/cache"
	"github.com/Rhaqim/buckt/internal/database"
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/mocks"
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/Rhaqim/buckt/internal/repository"
	"github.com/Rhaqim/buckt/internal/service"
	"github.com/Rhaqim/buckt/pkg/events"
	"github.com/Rhaqim/buckt/pkg/imageproc"
	"github.com/Rhaqim/buckt/pkg/logger"
	"github.com/Rhaqim/buckt/pkg/metrics"
	"github.com/Rhaqim/buckt/pkg/scan"
)

type Client struct {
	db *database.DB

	flatnameSpaces    bool
	silence           bool
	maxFileSize       int64
	maxTrashBatchSize int
	backendOpTimeout  time.Duration

	logger   domain.BucktLogger
	lruCache domain.LRUCache
	metrics  metrics.Recorder

	// backend, derivatives and imageProcessor support image-derivative
	// generation/retrieval, which lives at the Client layer (it composes GetFile
	// + backend writes). imageProcessor is never nil (defaults to the built-in).
	backend        Backend
	derivatives    []DerivativeSpec
	imageProcessor imageproc.Processor

	fileService   domain.FileService
	folderService domain.FolderService

	// sweeperStop cancels the optional background expiry sweeper (WithExpirySweeper).
	// Nil when no sweeper is running. Called by Close.
	sweeperStop func()
}

// New initializes a new Buckt client with the provided configuration options.
// It accepts a Config struct and a variadic number of ConfigFunc options as arguments and returns a pointer to the initialized Buckt client.
//
// Parameters:
// - conf: A Config struct containing the configuration options for the Buckt client.
// - opts: A variadic number of ConfigFunc options to customize the BucktConfig.
//
// Returns:
// - A pointer to the initialized Buckt Client.
// - An error if the Buckt client could not be created.
func New(conf Config, opts ...ConfigFunc) (*Client, error) {
	for _, opt := range opts {
		if opt != nil {
			opt(&conf)
		}
	}

	logConf := conf.Log
	bucktLog := logger.NewLogger(logConf.LogFile, logConf.LogTerminal, logConf.Silence, logger.WithLogger(logConf.Logger))
	bucktLog.Info("🚀 Starting Buckt")

	// Initialize database
	dbConf := conf.DB
	db, err := database.NewDB(dbConf.Database, dbConf.Driver, bucktLog, logConf.Silence, dbConf.TablePrefix)
	if err != nil {
		return nil, bucktLog.WrapErrorf("failed to initialize database", err)
	}

	// Migrate the database — fail fast on schema errors so we don't run
	// against an incompatible schema and corrupt data.
	if err = db.Migrate(conf.Backend.MigrationEnabled); err != nil {
		return nil, bucktLog.WrapErrorf("failed to migrate database", err)
	}

	// Initialize cache
	cacheManager, lruCache := initializeCache(conf.Cache, bucktLog)

	// Initialise Backend. In migration mode, back the resumable bulk copy with a
	// DB-persisted state store (the MigrationModel table, created by db.Migrate
	// above) so an interrupted MigrateAll resumes without re-scanning the target.
	var migrationStore domain.MigrationStateStore
	if conf.Backend.MigrationEnabled {
		migrationStore = repository.NewMigrationStateStore(db.DB)
	}
	backend := resolveBackend(conf.MediaDir, conf.Backend, bucktLog, lruCache, conf.Metrics, migrationStore)

	// Max file size: 0 means no limit (backward compatible)
	maxFileSize := conf.MaxFileSize

	// Trash batch limit: 0 -> default, negative -> unlimited (escape hatch)
	maxTrashBatch := conf.MaxTrashBatchSize
	if maxTrashBatch == 0 {
		maxTrashBatch = DefaultMaxTrashBatchSize
	}

	// Backend op timeout: 0 -> default, negative -> disabled
	backendTimeout := conf.BackendOpTimeout
	if backendTimeout == 0 {
		backendTimeout = DefaultBackendOpTimeout
	}

	// Image processor for derivatives: default to the built-in pure-Go one.
	imageProcessor := conf.ImageProcessor
	if imageProcessor == nil {
		imageProcessor = imageproc.Default()
	}

	// Initialize the app services
	folderService, fileService := newAppServices(
		conf.FlatNameSpaces,
		maxFileSize,
		maxTrashBatch,
		backendTimeout,
		db,
		bucktLog,
		cacheManager,
		backend,
		newEmitter(conf.EventHandlers, bucktLog),
		conf.Dedup,
		conf.Scanner,
	)

	// Initialize the Buckt instance
	buckt := &Client{
		db:                db,
		logger:            bucktLog,
		lruCache:          lruCache,
		metrics:           conf.Metrics,
		backend:           backend,
		derivatives:       conf.Derivatives,
		imageProcessor:    imageProcessor,
		flatnameSpaces:    conf.FlatNameSpaces,
		silence:           logConf.Silence,
		maxFileSize:       maxFileSize,
		maxTrashBatchSize: maxTrashBatch,
		backendOpTimeout:  backendTimeout,
		fileService:       fileService,
		folderService:     folderService,
	}

	// Optional background expiry sweeper (opt-in via WithExpirySweeper).
	if conf.ExpirySweepInterval > 0 {
		buckt.startExpirySweeper(conf.ExpirySweepInterval)
	}

	bucktLog.Info("✅ Buckt initialized")

	return buckt, nil
}

// Default initializes a new Buckt Client with default configuration options.
// It accepts a variadic number of ConfigFunc options to customize the BucktConfig.
//
// The default configuration includes:
// - LogConfig with LogTerminal set to true, LogFile set to "logs", and Debug set to true.
// - MediaDir set to "media".
// - FlatNameSpaces set to true.
//
// Parameters:
// - opts: A variadic number of ConfigFunc options to customize the BucktConfig.
//
// Returns:
// - A pointer to the initialized Buckt Client.
// - An error if the Buckt Client could not be created.
func Default(opts ...ConfigFunc) (*Client, error) {
	bucktOpts := Config{
		Log:            LogConfig{LogTerminal: true, Silence: true},
		MediaDir:       "media",
		FlatNameSpaces: true,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(&bucktOpts)
		}
	}

	return New(bucktOpts)
}

// Close releases the Buckt instance's resources: it closes the LRU cache and
// the database connection, and returns any error from closing the database
// (external connections passed via WithDB are left open). Adding this return
// value is source-compatible — existing `defer client.Close()` and
// `client.Close()` call sites keep working.
func (b *Client) Close() error {
	if b.sweeperStop != nil {
		b.sweeperStop()
	}
	b.lruCache.Close()
	return b.db.Close()
}

// MaxFileSize returns the configured maximum file size in bytes. A return
// value of 0 means no limit is enforced. Use this to surface the configured
// limit to HTTP handlers so they can reject oversized uploads early.
func (b *Client) MaxFileSize() int64 {
	return b.maxFileSize
}

// CacheStats returns the file cache hit and miss counts. Reads served from the
// cache issue no backend Get, so a high hit ratio directly reduces backend read
// operations (e.g. Cloudflare R2 Class B operations).
func (b *Client) CacheStats() (hits, misses uint64) {
	return b.lruCache.Hits(), b.lruCache.Misses()
}

// BackendName returns a human-readable name for the active storage backend —
// e.g. "local", "s3", or, in migration mode, "local->s3" (source->target).
func (b *Client) BackendName() string {
	return b.backend.Name()
}

// MetricsSnapshot returns the per-backend, per-operation metrics collected so
// far, plus true, when metrics were configured with a *metrics.Collector via
// WithMetrics. It returns (nil, false) when no metrics are configured or a
// custom (non-Collector) recorder was supplied — read those directly instead.
func (b *Client) MetricsSnapshot() (map[string]map[string]metrics.Stat, bool) {
	if c, ok := b.metrics.(*metrics.Collector); ok {
		return c.Snapshot(), true
	}
	return nil, false
}

// StorageBytes returns the total size, in bytes, of all stored files. Trashed
// files are included because their blobs still occupy backend storage until they
// are permanently deleted. This is the figure object stores like R2 bill for
// storage. Pass a per-request context to bound the query.
func (b *Client) StorageBytes(ctx context.Context) (int64, error) {
	var total int64
	// Include derivatives_bytes: generated image derivatives occupy backend
	// storage but are not tracked as files, so summing file sizes alone would
	// undercount.
	if err := b.db.WithContext(ctx).Model(&model.FileModel{}).
		Select("COALESCE(SUM(size), 0) + COALESCE(SUM(derivatives_bytes), 0)").Scan(&total).Error; err != nil {
		return 0, b.logger.WrapError("failed to compute storage bytes", err)
	}
	return total, nil
}

/* Folder Methods */

// NewFolder creates a new folder for a user within a specified parent folder.
//
// Parameters:
//   - user_id: The ID of the user creating the folder.
//   - parent_id: The ID of the parent folder where the new folder will be created.
//   - folder_name: The name of the new folder.
//   - description: A description of the new folder.
//
// Returns:
//   - The ID of the newly created folder.
//   - An error if the operation fails.
func (b *Client) NewFolder(user_id string, parent_id string, folder_name string, description string) (new_folder_id string, err error) {
	return b.NewFolderContext(context.Background(), user_id, parent_id, folder_name, description)
}

// ListFolders retrieves a list of folders for a given folder.
//
// Parameters:
//   - folder_id: The ID of the folder to retrieve.
//
// Returns:
//   - []model.FolderModel: A list of folders.
//   - error: An error if the folder could not be retrieved.
func (b *Client) ListFolders(folder_id string) ([]model.FolderModel, error) {
	return b.ListFoldersContext(context.Background(), folder_id)
}

// GetFolderWithContent retrieves a folder and its content.
//
// Parameters:
//   - user_id: The ID of the user who owns the folder.
//   - folder_id: The ID of the folder to retrieve the content for.
//
// Returns:
//   - *model.FolderModel: The folder model containing the folder content.
//   - error: An error if the folder content could not be retrieved.
func (b *Client) GetFolderWithContent(user_id, folder_id string) (*model.FolderModel, error) {
	return b.GetFolderWithContentContext(context.Background(), user_id, folder_id)
}

// MoveFolder moves a folder to a new parent folder.
//
// Parameters:
//   - user_id: The ID of the user performing the operation.
//   - folder_id: The ID of the folder to be moved.
//   - new_parent_id: The ID of the new parent folder.
//
// Returns:
//   - error: An error if the operation fails, otherwise nil.
func (b *Client) MoveFolder(user_id, folder_id string, new_parent_id string) error {
	return b.MoveFolderContext(context.Background(), user_id, folder_id, new_parent_id)
}

// RenameFolder renames a folder.
//
// Parameters:
//   - user_id: The ID of the user performing the operation.
//   - folder_id: The ID of the folder to be renamed.
//   - new_name: The new name for the folder.
//
// Returns:
//   - error: An error if the operation fails, otherwise nil.
func (b *Client) RenameFolder(user_id, folder_id string, new_name string) error {
	return b.RenameFolderContext(context.Background(), user_id, folder_id, new_name)
}

// DeleteFolder soft deletes a folder with the given folder_id using the folderService.
// It returns an error if the deletion fails.
//
// Parameters:
//   - folder_id: The ID of the folder to be deleted.
//
// Returns:
//   - error: An error if the deletion fails, otherwise nil.
func (b *Client) DeleteFolder(folder_id string) (string, error) {
	return b.DeleteFolderContext(context.Background(), folder_id)
}

// DeleteFolderPermanently deletes a folder permanently for a given user.
// It takes the user ID and folder ID as parameters and returns an error if the operation fails.
//
// Parameters:
//   - user_id: The ID of the user who owns the folder.
//   - folder_id: The ID of the folder to be deleted.
//
// Returns:
//   - error: An error object if the deletion fails, otherwise nil.
func (b *Client) DeleteFolderPermanently(user_id, folder_id string) (string, error) {
	return b.DeleteFolderPermanentlyContext(context.Background(), user_id, folder_id)
}

// GetTrashFolder returns the user's trash folder with its contents preloaded.
//
// Parameters:
//   - user_id: The ID of the user.
//
// Returns:
//   - *model.FolderModel: The trash folder model with its contents.
//   - error: An error if the trash folder could not be retrieved.
func (b *Client) GetTrashFolder(user_id string) (*model.FolderModel, error) {
	return b.GetTrashFolderContext(context.Background(), user_id)
}

/* File Methods */

// UploadFile uploads a file to the specified user's bucket.
//
// Parameters:
//   - user_id: The ID of the user who owns the bucket.
//   - parent_id: The ID of the parent directory where the file will be uploaded.
//   - file_name: The name of the file to be uploaded.
//   - content_type: The MIME type of the file.
//   - file_data: The byte slice containing the file data.
//
// Returns:
//   - string: The ID of the newly created file.
//   - error: An error if the file upload fails, otherwise nil.
func (b *Client) UploadFile(user_id string, parent_id string, file_name string, content_type string, file_data []byte) (string, error) {
	return b.UploadFileContext(context.Background(), user_id, parent_id, file_name, content_type, file_data)
}

// UploadFileFromReader uploads a file to the specified user's bucket from an io.Reader.
//
// Parameters:
//   - user_id: The ID of the user who owns the bucket.
//   - parent_id: The ID of the parent directory where the file will be uploaded.
//   - file_name: The name of the file to be uploaded.
//   - content_type: The MIME type of the file.
//   - file_data: An io.Reader containing the file data.
//
// Returns:
//   - string: The ID of the newly created file.
//   - error: An error if the file upload fails, otherwise nil.
func (b *Client) UploadFileFromReader(user_id string, parent_id string, file_name string, content_type string, file_data io.Reader) (string, error) {
	return b.UploadFileFromReaderContext(context.Background(), user_id, parent_id, file_name, content_type, file_data)
}

// GetFile retrieves a file based on the provided file ID.
// It returns the file data and an error, if any occurred during the retrieval process.
//
// Parameters:
//   - file_id: A string representing the unique identifier of the file to be retrieved.
//
// Returns:
//   - *model.FileModel: The file data.
//   - error: An error object if an error occurred, otherwise nil.
func (b *Client) GetFile(file_id string) (*model.FileModel, error) {
	return b.GetFileContext(context.Background(), file_id)
}

// GetFileStream retrieves a file stream based on the provided file ID.
// It returns the file data and an error, if any occurred during the retrieval process.
//
// Parameters:
//   - file_id: A string representing the unique identifier of the file to be retrieved.
//
// Returns:
//   - *model.FileModel: The file structure containing metadata.
//   - io.ReadCloser: An io.ReadCloser object representing the file stream
//   - error: An error object if an error occurred, otherwise nil.
//
// Note: The caller is responsible for closing the file stream after reading.
func (b *Client) GetFileStream(file_id string) (*model.FileModel, io.ReadCloser, error) {
	return b.GetFileStreamContext(context.Background(), file_id)
}

// ListFiles retrieves a list of files for a given folder.
//
// Parameters:
//   - folder_id: The ID of the folder to retrieve.
//
// Returns:
//
//   - []model.FileModel: A list of files.
//   - error: An error if the folder could not be retrieved.
func (b *Client) ListFiles(folder_id string) ([]model.FileModel, error) {
	return b.ListFilesContext(context.Background(), folder_id)
}

// ListFilesMetadata retrieves a list of files' metadata for a given folder.
//
// Parameters:
//   - folder_id: The ID of the folder to retrieve.
//
// Returns:
//
//   - []model.FileModel: A list of files' metadata.
//   - error: An error if the folder could not be retrieved.
func (b *Client) ListFilesMetadata(folder_id string) ([]model.FileModel, error) {
	return b.ListFilesMetadataContext(context.Background(), folder_id)
}

// MoveFile moves a file to a new parent directory.
//
// Parameters:
//   - file_id: The ID of the file to be updated.
//   - new_parent_id: The new parent directory for the file.
//
// Returns:
//   - error: An error if the update operation fails, otherwise nil.
func (b *Client) MoveFile(file_id string, new_parent_id string) error {
	return b.MoveFileContext(context.Background(), file_id, new_parent_id)
}

// DeleteFile deletes a file associated with the given user ID and file ID.
// It returns an error if the deletion fails.
//
// Parameters:
//   - file_id: The ID of the file to be deleted.
//
// Returns:
//   - error: An error if the file deletion fails, otherwise nil.
func (b *Client) DeleteFile(file_id string) (string, error) {
	return b.DeleteFileContext(context.Background(), file_id)
}

// DeleteFilePermanently deletes a file associated with the given user ID and file ID.
// It returns an error if the deletion fails.
//
// Parameters:
//   - file_id: The ID of the file to be deleted.
//
// Returns:
//   - error: An error if the file deletion fails, otherwise nil.
func (b *Client) DeleteFilePermanently(file_id string) (string, error) {
	return b.DeleteFilePermanentlyContext(context.Background(), file_id)
}

// RestoreFile moves a trashed file back to its original location (or the user's
// root folder if that location no longer exists) and removes it from trash.
//
// Parameters:
//   - user_id: The ID of the user who owns the file.
//   - file_id: The ID of the trashed file to restore.
//
// Returns:
//   - error: An error if the restore fails, otherwise nil.
func (b *Client) RestoreFile(user_id, file_id string) error {
	return b.RestoreFileContext(context.Background(), user_id, file_id)
}

// RestoreFolder moves a trashed folder (and its subtree) back to its original
// location (or the user's root folder if that location no longer exists) and
// removes it from trash.
//
// Parameters:
//   - user_id: The ID of the user who owns the folder.
//   - folder_id: The ID of the trashed folder to restore.
//
// Returns:
//   - error: An error if the restore fails, otherwise nil.
func (b *Client) RestoreFolder(user_id, folder_id string) error {
	return b.RestoreFolderContext(context.Background(), user_id, folder_id)
}

/* Contextual Folder Methods */

// NewFolderContext creates a new folder for a user within a specified parent folder.
//
// Parameters:
//   - ctx: The context for the operation.
//   - user_id: The ID of the user creating the folder.
//   - parent_id: The ID of the parent folder where the new folder will be created.
//   - folder_name: The name of the new folder.
//   - description: A description of the new folder.
//
// Returns:
//   - The ID of the newly created folder.
//   - An error if the operation fails.
func (b *Client) NewFolderContext(ctx context.Context, user_id string, parent_id string, folder_name string, description string) (new_folder_id string, err error) {
	return b.folderService.CreateFolder(ctx, user_id, parent_id, folder_name, description)
}

// ListFoldersContext retrieves a list of folders for a given folder.
//
// Parameters:
//   - ctx: The context for the operation.
//   - folder_id: The ID of the folder to retrieve.
//
// Returns:
//   - []model.FolderModel: A list of folders.
//   - error: An error if the folder could not be retrieved.
func (b *Client) ListFoldersContext(ctx context.Context, folder_id string) ([]model.FolderModel, error) {
	return b.folderService.GetFolders(ctx, folder_id)
}

// GetFolderWithContentContext retrieves a folder and its content.
//
// Parameters:
//   - ctx: The context for the operation.
//   - user_id: The ID of the user who owns the folder.
//   - folder_id: The ID of the folder to retrieve the content for.
//
// Returns:
//   - *model.FolderModel: The folder model containing the folder content.
//   - error: An error if the folder content could not be retrieved.
func (b *Client) GetFolderWithContentContext(ctx context.Context, user_id, folder_id string) (*model.FolderModel, error) {
	return b.folderService.GetFolder(ctx, user_id, folder_id)
}

// MoveFolderContext moves a folder to a new parent folder.
//
// Parameters:
//   - ctx: The context for the operation.
//   - user_id: The ID of the user performing the operation.
//   - folder_id: The ID of the folder to be moved.
//   - new_parent_id: The ID of the new parent folder.
//
// Returns:
//   - error: An error if the operation fails, otherwise nil.
func (b *Client) MoveFolderContext(ctx context.Context, user_id, folder_id string, new_parent_id string) error {
	return b.folderService.MoveFolder(ctx, folder_id, new_parent_id)
}

// RenameFolderContext renames a folder.
//
// Parameters:
//   - ctx: The context for the operation.
//   - user_id: The ID of the user performing the operation.
//   - folder_id: The ID of the folder to be renamed.
//   - new_name: The new name for the folder.
//
// Returns:
//   - error: An error if the operation fails, otherwise nil.
func (b *Client) RenameFolderContext(ctx context.Context, user_id, folder_id string, new_name string) error {
	return b.folderService.RenameFolder(ctx, user_id, folder_id, new_name)
}

// DeleteFolderContext soft deletes a folder with the given folder_id using the folderService.
// It returns an error if the deletion fails.
//
// Parameters:
//   - ctx: The context for the operation.
//   - folder_id: The ID of the folder to be deleted.
//
// Returns:
//   - error: An error if the deletion fails, otherwise nil.
func (b *Client) DeleteFolderContext(ctx context.Context, folder_id string) (string, error) {
	return b.folderService.DeleteFolder(ctx, folder_id)
}

// DeleteFolderPermanentlyContext deletes a folder permanently for a given user.
// It takes the user ID and folder ID as parameters and returns an error if the operation fails.
//
// Parameters:
//   - ctx: The context for the operation.
//   - user_id: The ID of the user who owns the folder.
//   - folder_id: The ID of the folder to be deleted.
//
// Returns:
//   - error: An error object if the deletion fails, otherwise nil.
func (b *Client) DeleteFolderPermanentlyContext(ctx context.Context, user_id, folder_id string) (string, error) {

	// If flatnameSpaces is enabled, we soft delete the folder
	if b.flatnameSpaces {
		return b.folderService.DeleteFolder(ctx, folder_id)
	}

	return b.folderService.ScrubFolder(ctx, user_id, folder_id)
}

// GetTrashFolderContext returns the user's trash folder with its contents preloaded.
func (b *Client) GetTrashFolderContext(ctx context.Context, user_id string) (*model.FolderModel, error) {
	return b.folderService.GetTrashFolder(ctx, user_id)
}

// RestoreFolderContext moves a trashed folder (and its subtree) back to the
// folder it was in before it was trashed, or the user's root folder if that
// original location no longer exists. Blobs are physically moved in
// nested-namespace mode.
//
// Parameters:
//   - ctx: The context for the operation.
//   - user_id: The ID of the user who owns the folder.
//   - folder_id: The ID of the trashed folder to restore.
//
// Returns:
//   - error: An error if the restore fails, otherwise nil.
func (b *Client) RestoreFolderContext(ctx context.Context, user_id, folder_id string) error {
	return b.folderService.RestoreFolder(ctx, user_id, folder_id)
}

/* File Methods */

// UploadFileContext uploads a file to the specified user's bucket.
//
// Parameters:
//   - ctx: The context for the operation.
//   - user_id: The ID of the user who owns the bucket.
//   - parent_id: The ID of the parent directory where the file will be uploaded.
//   - file_name: The name of the file to be uploaded.
//   - content_type: The MIME type of the file.
//   - file_data: The byte slice containing the file data.
//
// Returns:
//   - string: The ID of the newly created file.
//   - error: An error if the file upload fails, otherwise nil.
func (b *Client) UploadFileContext(ctx context.Context, user_id string, parent_id string, file_name string, content_type string, file_data []byte) (string, error) {
	return b.fileService.CreateFile(ctx, user_id, parent_id, file_name, content_type, file_data)
}

// UploadFileFromReaderContext uploads a file to the specified user's bucket from an io.Reader.
//
// Parameters:
//   - ctx: The context for the operation.
//   - user_id: The ID of the user who owns the bucket.
//   - parent_id: The ID of the parent directory where the file will be uploaded.
//   - file_name: The name of the file to be uploaded.
//   - content_type: The MIME type of the file.
//   - file_data: An io.Reader containing the file data.
//
// Returns:
//   - string: The ID of the newly created file.
//   - error: An error if the file upload fails, otherwise nil.
func (b *Client) UploadFileFromReaderContext(ctx context.Context, user_id string, parent_id string, file_name string, content_type string, file_data io.Reader) (string, error) {
	// Try to use Seeker for efficiency if available
	if seeker, ok := file_data.(io.Seeker); ok {
		fileSize, err := seeker.Seek(0, io.SeekEnd)
		if err != nil {
			return "", err
		}
		// Reject before allocating if we know the size upfront
		if b.maxFileSize > 0 && fileSize > b.maxFileSize {
			return "", fmt.Errorf("file size %d exceeds maximum allowed size %d bytes", fileSize, b.maxFileSize)
		}
		// Guard against int64 -> int overflow on 32-bit platforms
		if fileSize < 0 || fileSize > int64(math.MaxInt) {
			return "", fmt.Errorf("file size %d out of range for int allocation", fileSize)
		}
		if _, err = seeker.Seek(0, io.SeekStart); err != nil {
			return "", err
		}

		file_bytes := make([]byte, int(fileSize))
		if _, err = io.ReadFull(file_data, file_bytes); err != nil {
			return "", err
		}
		return b.fileService.CreateFile(ctx, user_id, parent_id, file_name, content_type, file_bytes)
	}

	// Fallback: read with bounded reader for non-seekable streams
	reader := file_data
	if b.maxFileSize > 0 {
		reader = io.LimitReader(file_data, b.maxFileSize+1) // +1 to detect overflow
	}
	file_bytes, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	if b.maxFileSize > 0 && int64(len(file_bytes)) > b.maxFileSize {
		return "", fmt.Errorf("file size exceeds maximum allowed size %d bytes", b.maxFileSize)
	}

	return b.fileService.CreateFile(ctx, user_id, parent_id, file_name, content_type, file_bytes)
}

// GetFileContext retrieves a file based on the provided file ID.
// It returns the file data and an error, if any occurred during the retrieval process.
//
// Parameters:
//   - ctx: The context for the operation.
//   - file_id: A string representing the unique identifier of the file to be retrieved.
//
// Returns:
//   - *model.FileModel: The file data.
//   - error: An error object if an error occurred, otherwise nil.
func (b *Client) GetFileContext(ctx context.Context, file_id string) (*model.FileModel, error) {
	return b.fileService.GetFile(ctx, file_id)
}

// GetFileStreamContext retrieves a file stream based on the provided file ID.
// It returns the file data and an error, if any occurred during the retrieval process.
//
// Parameters:
//   - ctx: The context for the operation.
//   - file_id: A string representing the unique identifier of the file to be retrieved.
//
// Returns:
//   - *model.FileModel: The file structure containing metadata.
//   - io.ReadCloser: An io.ReadCloser object representing the file stream
//   - error: An error object if an error occurred, otherwise nil.
//
// Note: The caller is responsible for closing the file stream after reading.
func (b *Client) GetFileStreamContext(ctx context.Context, file_id string) (*model.FileModel, io.ReadCloser, error) {
	return b.fileService.GetFileStream(ctx, file_id)
}

// ListFilesContext retrieves a list of files for a given folder.
//
// Parameters:
//   - ctx: The context for the operation.
//   - folder_id: The ID of the folder to retrieve.
//
// Returns:
//
//   - []model.FileModel: A list of files.
//   - error: An error if the folder could not be retrieved.
func (b *Client) ListFilesContext(ctx context.Context, folder_id string) ([]model.FileModel, error) {
	return b.fileService.GetFiles(ctx, folder_id)
}

// ListFilesMetadataContext retrieves a list of files' metadata for a given folder.
//
// Parameters:
//   - ctx: The context for the operation.
//   - folder_id: The ID of the folder to retrieve.
//
// Returns:
//
//   - []model.FileModel: A list of files' metadata.
//   - error: An error if the folder could not be retrieved.
func (b *Client) ListFilesMetadataContext(ctx context.Context, folder_id string) ([]model.FileModel, error) {
	return b.fileService.GetFilesMetadata(ctx, folder_id)
}

// MoveFileContext moves a file to a new parent directory.
//
// Parameters:
//   - ctx: The context for the operation.
//   - file_id: The ID of the file to be updated.
//   - new_parent_id: The new parent directory for the file.
//
// Returns:
//   - error: An error if the update operation fails, otherwise nil.
func (b *Client) MoveFileContext(ctx context.Context, file_id string, new_parent_id string) error {
	return b.fileService.MoveFile(ctx, file_id, new_parent_id)
}

// DeleteFileContext deletes a file associated with the given user ID and file ID.
// It returns an error if the deletion fails.
//
// Parameters:
//   - ctx: The context for the operation.
//   - file_id: The ID of the file to be deleted.
//
// Returns:
//   - error: An error if the file deletion fails, otherwise nil.
func (b *Client) DeleteFileContext(ctx context.Context, file_id string) (string, error) {
	return b.fileService.DeleteFile(ctx, file_id)
}

// DeleteFilePermanentlyContext deletes a file associated with the given user ID and file ID.
// It returns an error if the deletion fails.
//
// Parameters:
//   - ctx: The context for the operation.
//   - file_id: The ID of the file to be deleted.
//
// Returns:
//   - error: An error if the file deletion fails, otherwise nil.
func (b *Client) DeleteFilePermanentlyContext(ctx context.Context, file_id string) (string, error) {
	parentID, err := b.fileService.ScrubFile(ctx, file_id)
	if err != nil {
		return parentID, err
	}
	// Best-effort: remove any generated derivatives. They are keyed by file id
	// and not tracked as files, so nothing else cleans them up.
	if len(b.derivatives) > 0 {
		if derr := b.backend.DeleteFolder(ctx, derivativeDir(file_id)); derr != nil {
			b.logger.Warn("failed to delete derivatives for scrubbed file " + file_id + ": " + derr.Error())
		}
	}
	return parentID, nil
}

// RestoreFileContext moves a trashed file back to the folder it was in before
// it was trashed, or the user's root folder if that original location no longer
// exists. The blob is physically moved in nested-namespace mode.
//
// Parameters:
//   - ctx: The context for the operation.
//   - user_id: The ID of the user who owns the file.
//   - file_id: The ID of the trashed file to restore.
//
// Returns:
//   - error: An error if the restore fails, otherwise nil.
func (b *Client) RestoreFileContext(ctx context.Context, user_id, file_id string) error {
	return b.fileService.RestoreFile(ctx, user_id, file_id)
}

// SetFileMetadata replaces the arbitrary key/value metadata stored on a file.
// buckt does not interpret it — use it for resource links (e.g. order IDs),
// AI-extracted tags, OCR text, etc. Passing nil or an empty map clears it.
//
// Parameters:
//   - file_id: The ID of the file.
//   - metadata: The key/value map to store (replaces any existing metadata).
//
// Returns:
//   - error: An error if the file is not found or the update fails.
func (b *Client) SetFileMetadata(file_id string, metadata map[string]string) error {
	return b.SetFileMetadataContext(context.Background(), file_id, metadata)
}

// GetFileMetadata returns the metadata stored on a file (nil when none is set).
func (b *Client) GetFileMetadata(file_id string) (map[string]string, error) {
	return b.GetFileMetadataContext(context.Background(), file_id)
}

// SetFileMetadataContext is SetFileMetadata with an explicit context.
func (b *Client) SetFileMetadataContext(ctx context.Context, file_id string, metadata map[string]string) error {
	return b.fileService.SetMetadata(ctx, file_id, metadata)
}

// GetFileMetadataContext is GetFileMetadata with an explicit context.
func (b *Client) GetFileMetadataContext(ctx context.Context, file_id string) (map[string]string, error) {
	return b.fileService.GetMetadata(ctx, file_id)
}

/* Image derivatives */

// isImage reports whether the content type is an image, so derivative
// generation is attempted (the configured processor decides which encodings it
// actually supports). Non-images are skipped, making GenerateDerivatives safe
// to call for every upload.
func isImage(contentType string) bool {
	return len(contentType) >= 6 && contentType[:6] == "image/"
}

// derivativeDir is the backend key prefix holding a file's derivatives. Keying
// by file id (not path) means moving or trashing the file never orphans them.
func derivativeDir(file_id string) string { return ".buckt_derivatives/" + file_id }

func derivativeKey(file_id, name string) string { return derivativeDir(file_id) + "/" + name }

// GenerateDerivatives produces the configured resized variants (see
// WithImageDerivatives) for a file and stores them. It's a no-op when no
// variants are configured or the file isn't a supported image (JPEG/PNG), so it
// is safe to call for every upload. Resizing is CPU-bound — call this from a
// file.uploaded event handler (ideally a background worker) rather than in the
// request path.
func (b *Client) GenerateDerivatives(file_id string) error {
	return b.GenerateDerivativesContext(context.Background(), file_id)
}

// GetDerivative returns the bytes and detected content type of a previously
// generated variant, or ErrNotFound if it hasn't been generated.
func (b *Client) GetDerivative(file_id, name string) ([]byte, string, error) {
	return b.GetDerivativeContext(context.Background(), file_id, name)
}

// GenerateDerivativesContext is GenerateDerivatives with an explicit context.
func (b *Client) GenerateDerivativesContext(ctx context.Context, file_id string) error {
	if len(b.derivatives) == 0 {
		return nil
	}

	file, err := b.fileService.GetFile(ctx, file_id)
	if err != nil {
		return err
	}
	if !isImage(file.ContentType) {
		return nil
	}

	var totalBytes int64
	for _, spec := range b.derivatives {
		out, _, err := b.imageProcessor.Resize(file.Data, spec.MaxWidth, spec.Format)
		if err != nil {
			return b.logger.WrapError("failed to resize derivative "+spec.Name, err)
		}
		if err := b.backend.Put(ctx, derivativeKey(file_id, spec.Name), out); err != nil {
			return b.logger.WrapError("failed to store derivative "+spec.Name, err)
		}
		totalBytes += int64(len(out))
	}

	// Record the total derivative size so StorageBytes accounts for it. This
	// replaces any previous value (regenerating overwrites the variants).
	if err := b.db.WithContext(ctx).Model(&model.FileModel{}).
		Where("id = ?", file.ID).Update("derivatives_bytes", totalBytes).Error; err != nil {
		return b.logger.WrapError("failed to record derivative size", err)
	}
	return nil
}

// GetDerivativeContext is GetDerivative with an explicit context.
func (b *Client) GetDerivativeContext(ctx context.Context, file_id, name string) ([]byte, string, error) {
	data, err := b.backend.Get(ctx, derivativeKey(file_id, name))
	if err != nil {
		return nil, "", fmt.Errorf("derivative %q not found: %w", name, ErrNotFound)
	}
	return data, http.DetectContentType(data), nil
}

/* Migration */

// MigrateAll starts a background copy of every stored object from the source
// backend to the target backend. It works only when the Client was created with
// WithMigration; otherwise it returns ErrBackendUnavailable. It returns as soon
// as the copy is scheduled — poll MigrationStatus for progress. Objects already
// present in the target are skipped, so it is safe to call more than once.
//
// This is the bulk counterpart to the automatic behaviour of migration mode
// (every write is mirrored to the target, and reads lazily copy forward): use
// it to move pre-existing files that predate the cutover.
func (b *Client) MigrateAll(ctx context.Context) error {
	m, ok := b.backend.(domain.MigratableBackend)
	if !ok {
		return fmt.Errorf("migration is not enabled — create the client with WithMigration: %w", ErrBackendUnavailable)
	}
	return m.MigrateAll(ctx)
}

// MigrationStatus reports how many objects have been processed by the most
// recent MigrateAll and the total scheduled. ok is false when the Client was not
// created with WithMigration. When a migration finishes, completed == total —
// completed counts every processed object, whether it was copied, already
// present, or permanently failed after retries. Use MigrationFailures to find
// out how many of those permanently failed (successfully copied == completed -
// failed).
func (b *Client) MigrationStatus(ctx context.Context) (completed, total int64, ok bool) {
	m, isMigratable := b.backend.(domain.MigratableBackend)
	if !isMigratable {
		return 0, 0, false
	}
	copied, failed, tot := m.MigrationStatus(ctx)
	return copied + failed, tot, true
}

// MigrationFailures reports how many objects the most recent MigrateAll could
// not copy after retries. ok is false when the Client was not created with
// WithMigration. A non-zero count means the migration finished (completed ==
// total) but some objects were left behind — inspect the logs for which, and
// re-run MigrateAll to retry them once the underlying issue is fixed.
func (b *Client) MigrationFailures(ctx context.Context) (failed int64, ok bool) {
	m, isMigratable := b.backend.(domain.MigratableBackend)
	if !isMigratable {
		return 0, false
	}
	_, failed, _ = m.MigrationStatus(ctx)
	return failed, true
}

/* Expiry */

// SetFileExpiry sets the time at which a file is automatically, permanently
// deleted by PurgeExpired (and by the optional background sweeper). Passing the
// zero time.Time clears the expiry, making the file permanent again.
func (b *Client) SetFileExpiry(file_id string, at time.Time) error {
	return b.SetFileExpiryContext(context.Background(), file_id, at)
}

// SetFileExpiryContext is SetFileExpiry with an explicit context.
func (b *Client) SetFileExpiryContext(ctx context.Context, file_id string, at time.Time) error {
	if at.IsZero() {
		return b.fileService.SetExpiry(ctx, file_id, nil)
	}
	return b.fileService.SetExpiry(ctx, file_id, &at)
}

// SetFileTTL sets a file to expire ttl from now — the convenient form for temp
// files ("delete this in 1h"). A non-positive ttl clears the expiry.
func (b *Client) SetFileTTL(file_id string, ttl time.Duration) error {
	return b.SetFileTTLContext(context.Background(), file_id, ttl)
}

// SetFileTTLContext is SetFileTTL with an explicit context.
func (b *Client) SetFileTTLContext(ctx context.Context, file_id string, ttl time.Duration) error {
	if ttl <= 0 {
		return b.fileService.SetExpiry(ctx, file_id, nil)
	}
	return b.SetFileExpiryContext(ctx, file_id, time.Now().Add(ttl))
}

// PurgeExpired permanently deletes every file whose expiry has passed — blob,
// image derivatives, and metadata row — emitting a file.uploaded-style
// file.purged event for each (so event handlers can react). It returns the
// number purged. Safe to call repeatedly; call it from your own scheduler, or
// let WithExpirySweeper call it for you.
//
// Work is done in batches so a large backlog doesn't load every row at once.
// A file that fails to purge is logged and left for the next run; if an entire
// batch fails to make progress, PurgeExpired stops and returns an error rather
// than spinning.
func (b *Client) PurgeExpired(ctx context.Context) (purged int, err error) {
	const batch = 500
	for {
		if err := ctx.Err(); err != nil {
			return purged, err
		}

		files, ferr := b.fileService.FindExpired(ctx, time.Now(), batch)
		if ferr != nil {
			return purged, ferr
		}
		if len(files) == 0 {
			return purged, nil
		}

		progressed := 0
		for _, f := range files {
			if _, derr := b.DeleteFilePermanentlyContext(ctx, f.ID.String()); derr != nil {
				b.logger.Warn("failed to purge expired file " + f.ID.String() + ": " + derr.Error())
				continue
			}
			purged++
			progressed++
		}

		// Every file in this batch failed — stop rather than loop forever over
		// the same undeletable rows.
		if progressed == 0 {
			return purged, fmt.Errorf("failed to purge any of %d expired file(s): %w", len(files), ErrBackendUnavailable)
		}
		// A short final batch means there's nothing left to fetch.
		if len(files) < batch {
			return purged, nil
		}
	}
}

// startExpirySweeper launches the background ticker started by WithExpirySweeper.
func (b *Client) startExpirySweeper(interval time.Duration) {
	ctx, cancel := context.WithCancel(context.Background())
	b.sweeperStop = cancel
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if n, err := b.PurgeExpired(ctx); err != nil {
					b.logger.Warn("expiry sweep failed: " + err.Error())
				} else if n > 0 {
					b.logger.Infof("expiry sweep purged %d file(s)", n)
				}
			}
		}
	}()
}

/* Helper Methods */

func initializeCache(conf CacheConfig, bucktLog domain.BucktLogger) (domain.CacheManager, domain.LRUCache) {
	fileConf := conf.FileCacheConfig
	fileConf.Validate()

	lruCache, err := cache.NewFileCache(fileConf.NumCounters, fileConf.MaxCost, fileConf.BufferItems)
	if err != nil {
		// Best-effort fallback: the cache is optional, so we don't fail startup.
		// Log it (the one legitimate use of the logger — an error we deliberately
		// do not return to the caller).
		bucktLog.Warn("failed to initialize file cache, using no-op cache: " + err.Error())
		lruCache = mocks.NewNoopLRUCache()
	}
	bucktLog.Info("✅ Initialized file cache")

	if conf.Manager != nil {
		bucktLog.Info("✅ Using provided cache")
		return conf.Manager, lruCache
	}
	return mocks.NewNoopCache(), lruCache
}

// newEmitter returns a single events.Handler that fans an event out to every
// registered handler, isolating each behind panic recovery so a misbehaving
// handler can neither crash nor fail the originating operation. With no handlers
// it returns a no-op, so the service can always call emit unconditionally.
func newEmitter(handlers []events.Handler, log domain.BucktLogger) events.Handler {
	if len(handlers) == 0 {
		return func(context.Context, events.Event) {}
	}
	hs := make([]events.Handler, len(handlers))
	copy(hs, handlers)
	return func(ctx context.Context, e events.Event) {
		for _, h := range hs {
			dispatchEvent(ctx, h, e, log)
		}
	}
}

func dispatchEvent(ctx context.Context, h events.Handler, e events.Event, log domain.BucktLogger) {
	defer func() {
		if r := recover(); r != nil {
			log.Warn(fmt.Sprintf("buckt: event handler for %s panicked: %v", e.Type, r))
		}
	}()
	h(ctx, e)
}

func newAppServices(
	flatNameSpaces bool,
	maxFileSize int64,
	maxTrashBatch int,
	backendOpTimeout time.Duration,
	db *database.DB,
	logger domain.BucktLogger,
	cacheManager domain.CacheManager,
	activeBackend domain.FileBackend,
	emit events.Handler,
	dedup bool,
	scanner scan.Scanner,
) (domain.FolderService, domain.FileService) {
	// Initialize the stores
	folderRepository := repository.NewFolderRepository(db)
	fileRepository := repository.NewFileRepository(db)

	// initialize the services
	folderService := service.NewFolderService(logger, cacheManager, folderRepository, activeBackend, flatNameSpaces, maxTrashBatch, backendOpTimeout)
	fileService := service.NewFileService(logger, cacheManager, fileRepository, folderService, activeBackend, flatNameSpaces, maxFileSize, backendOpTimeout, emit, dedup, scanner)

	logger.Info("✅ Initialized app services")

	return folderService, fileService
}

func resolveBackend(
	mediaDir string,
	bc BackendConfig,
	log domain.BucktLogger,
	lru domain.LRUCache,
	rec metrics.Recorder,
	migrationStore domain.MigrationStateStore,
) Backend {
	// meter wraps a leaf backend so every operation is recorded. It is a no-op
	// when rec is nil. In migration mode the leaves (source/target) are metered
	// individually — never the composite — so per-backend counts stay distinct
	// and nothing is double-counted.
	meter := func(b Backend) Backend { return backend.NewMeteredBackend(b, rec) }

	if bc.MigrationEnabled {
		var source, target Backend

		// Fallback logic for source
		if bc.Source != nil {
			source = resolveIfPlaceholder(bc.Source, mediaDir, log, lru)
		} else {
			log.Warn("⚠️ Migration enabled but source backend missing — falling back to local as source")
			source = backend.NewLocalFileSystemService(log, mediaDir, lru)
		}

		// Fallback logic for target
		if bc.Target != nil {
			target = resolveIfPlaceholder(bc.Target, mediaDir, log, lru)
		} else {
			log.Warn("⚠️ Migration enabled but target backend missing — falling back to local as target")
			target = backend.NewLocalFileSystemService(log, mediaDir, lru)
		}

		// ensure both source and target are set and different
		if source == nil || target == nil {
			log.Errorf("❌ Migration enabled but one of the backends is nil — falling back to local")
			return meter(backend.NewLocalFileSystemService(log, mediaDir, lru))
		}

		// Compare by pointer identity — only reject if the caller passed the
		// exact same backend instance for both source and target. Two distinct
		// backends with the same Name() (e.g. two S3 buckets) are legitimate
		// migration targets and should not be rejected here. Users are responsible
		// for ensuring their source and target point to different underlying
		// storage locations.
		//
		// NOTE: this identity check runs on the UNWRAPPED backends; metering
		// below would otherwise produce two distinct wrapper instances and defeat
		// the comparison.
		if source == target {
			log.Errorf("❌ Migration enabled but source and target are the same instance — disabling migration and using source only")
			return meter(source)
		}

		log.Infof("🔄 Migration mode: %s → %s", source.Name(), target.Name())
		return backend.NewMigrationBackend(log, meter(source), meter(target), bc.MigrationConcurrency, migrationStore)
	}

	// Non-migration modes
	switch {
	case bc.Source != nil:
		return meter(resolveIfPlaceholder(bc.Source, mediaDir, log, lru))

	case bc.Target != nil:
		log.Warn("⚠️ Using target backend as primary because source is missing")
		return meter(resolveIfPlaceholder(bc.Target, mediaDir, log, lru))

	default:
		log.Warn("⚠️ No backend configured, falling back to local")
		return meter(backend.NewLocalFileSystemService(log, mediaDir, lru))
	}
}

// resolveIfPlaceholder checks if the backend is a PlaceholderBackend (e.g. from
// buckt.LocalBackend()) and replaces it with a real local backend instance.
// User-provided cloud backends pass through unchanged.
func resolveIfPlaceholder(b Backend, mediaDir string, log domain.BucktLogger, lru domain.LRUCache) Backend {
	if _, ok := b.(*domain.PlaceholderBackend); ok {
		return backend.NewLocalFileSystemService(log, mediaDir, lru)
	}
	return b
}
