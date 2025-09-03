// Description: Buckt is a simple file storage service that allows users to upload, download, and manage files and folders.
// It provides a simple API for managing files and folders, as well as a web interface for interacting with the service.
// Buckt supports multiple storage backends, including local storage and cloud storage providers.
// The service can be configured to use a specific storage backend, or it can be used as a standalone service with local storage.
// Buckt is built using Go and provides a simple and easy-to-use API for managing files and folders.
// It is designed to be lightweight and easy to deploy, making it ideal for small projects and personal use.
// The service is extensible and can be customized to support additional features and functionality.

package buckt

import (
	"io"

	"github.com/Rhaqim/buckt/internal/cache"
	"github.com/Rhaqim/buckt/internal/database"
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/Rhaqim/buckt/internal/repository"
	"github.com/Rhaqim/buckt/internal/service"
	"github.com/Rhaqim/buckt/pkg/logger"
)

type Client struct {
	db     *database.DB
	logger *logger.BucktLogger

	FlatnameSpaces bool
	Debug          bool

	lruCache domain.LRUCache

	fileService   domain.FileService
	folderService domain.FolderService
}

// New initializes a new Buckt instance with the provided configuration options.
// It accepts a BucktConfig struct as an argument and returns a pointer to the initialized Buckt instance.
//
// Parameters:
// - bucktOpts: A BucktConfig struct containing the configuration options for the Buckt instance.
//
// Returns:
// - A pointer to the initialized Buckt instance.
// - An error if the Buckt instance could not be created.
func New(conf Config, opts ...ConfigFunc) (*Client, error) {
	for _, opt := range opts {
		opt(&conf)
	}

	logConf := conf.Log

	bucktLog := logger.NewLogger(logConf.LogFile, logConf.LogTerminal, logConf.Debug, logger.WithLogger(logConf.Logger))
	bucktLog.Info("🚀 Starting Buckt")

	dbConf := conf.DB

	// Initialize database
	db, err := database.NewDB(dbConf.Database, dbConf.Driver, bucktLog, logConf.Debug)
	if err != nil {
		return nil, bucktLog.WrapErrorf("failed to initialize database", err)
	}

	// Migrate the database
	if err = db.Migrate(); err != nil {
		bucktLog.WrapErrorf("failed to migrate database", err)
	}

	// Initialize cache
	cacheManager, lruCache := initializeCache(conf.Cache, bucktLog)

	// Initialise Backend
	var backend domain.FileBackend = resolveBackend(conf.MediaDir, conf.Backend, bucktLog, lruCache)

	// Initialize the app services
	folderService, fileService := newAppServices(

		conf.FlatNameSpaces,
		bucktLog,
		db,
		cacheManager,
		backend,
	)

	// Initialize the Buckt instance
	buckt := &Client{
		db:             db,
		logger:         bucktLog,
		lruCache:       lruCache,
		FlatnameSpaces: conf.FlatNameSpaces,
		Debug:          logConf.Debug,
		fileService:    fileService,
		folderService:  folderService,
	}

	bucktLog.Info("✅ Buckt initialized")

	return buckt, nil
}

// Default initializes a new Buckt instance with default configuration options.
// It accepts a variadic number of ConfigFunc options to customize the BucktConfig.
//
// The default configuration includes:
// - LogConfig with LogTerminal set to true, LogFile set to "logs", and Debug set to true.
// - MediaDir set to "media".
// - StandaloneMode set to true.
// - FlatNameSpaces set to true.
//
// Parameters:
// - opts: A variadic number of ConfigFunc options to customize the BucktConfig.
//
// Returns:
// - A pointer to the initialized Buckt instance.
// - An error if the Buckt instance could not be created.
func Default(opts ...ConfigFunc) (*Client, error) {
	bucktOpts := Config{
		Log:            LogConfig{LogTerminal: true, Debug: true},
		MediaDir:       "media",
		FlatNameSpaces: true,
	}

	for _, opt := range opts {
		opt(&bucktOpts)
	}

	return New(bucktOpts)
}

// Close closes the Buckt instance.
// It closes the database connection.
func (b *Client) Close() {
	b.db.Close()
	b.lruCache.Close()
}

/* Folder Methods */

// NewFolder creates a new folder for a user within a specified parent folder.
// It takes the following parameters:
// - user_id: The ID of the user creating the folder.
// - parent_id: The ID of the parent folder where the new folder will be created.
// - folder_name: The name of the new folder.
// - description: A description of the new folder.
// It returns the ID of the newly created folder and an error if the operation fails.
func (b *Client) NewFolder(user_id string, parent_id string, folder_name string, description string) (new_folder_id string, err error) {
	return b.folderService.CreateFolder(user_id, parent_id, folder_name, description)
}

// ListFolders retrieves a list of folders for a given folder.
//
// Parameters:
//
//	folder_id - The ID of the folder to retrieve.
//
// Returns:
//
//	[]model.FolderModel - A list of folders.
//	error - An error if the folder could not be retrieved.
func (b *Client) ListFolders(folder_id string) ([]model.FolderModel, error) {
	return b.folderService.GetFolders(folder_id)
}

// GetFolderWithContent retrieves a folder and its content.
//
// Parameters:
//
//	user_id - The ID of the user who owns the folder.
//	folder_id - The ID of the folder to retrieve the content for.
//
// Returns:
//
//	*model.FolderModel - The folder model containing the folder content.
//	error - An error if the folder content could not be retrieved.
func (b *Client) GetFolderWithContent(user_id, folder_id string) (*model.FolderModel, error) {
	return b.folderService.GetFolder(user_id, folder_id)
}

// MoveFolder moves a folder to a new parent folder.
//
// Parameters:
//
//	user_id: The ID of the user performing the operation.
//	folder_id: The ID of the folder to be moved.
//	new_parent_id: The ID of the new parent folder.
//
// Returns:
//
//	error: An error if the operation fails, otherwise nil.
func (b *Client) MoveFolder(user_id, folder_id string, new_parent_id string) error {
	return b.folderService.MoveFolder(folder_id, new_parent_id)
}

// RenameFolder renames a folder.
//
// Parameters:
//
//	user_id: The ID of the user performing the operation.
//	folder_id: The ID of the folder to be renamed.
//	new_name: The new name for the folder.
//
// Returns:
//
//	error: An error if the operation fails, otherwise nil.
func (b *Client) RenameFolder(user_id, folder_id string, new_name string) error {
	return b.folderService.RenameFolder(user_id, folder_id, new_name)
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
	return b.folderService.DeleteFolder(folder_id)
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

	// If flatnameSpaces is enabled, we soft delete the folder
	if b.FlatnameSpaces {
		return b.folderService.DeleteFolder(folder_id)
	}

	return b.folderService.ScrubFolder(user_id, folder_id)
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
	return b.fileService.CreateFile(user_id, parent_id, file_name, content_type, file_data)
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
	return b.fileService.GetFile(file_id)
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
	return b.fileService.GetFileStream(file_id)
}

// ListFiles retrieves a list of files for a given folder.
//
// Parameters:
//
//	folder_id - The ID of the folder to retrieve.
//
// Returns:
//
//	[]model.FileModel - A list of files.
//	error - An error if the folder could not be retrieved.
func (b *Client) ListFiles(folder_id string) ([]model.FileModel, error) {
	return b.fileService.GetFiles(folder_id)
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
	return b.fileService.MoveFile(file_id, new_parent_id)
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
	return b.fileService.DeleteFile(file_id)
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
	return b.fileService.ScrubFile(file_id)
}

/* Migration */

/* Helper Methods */

func initializeCache(conf CacheConfig, bucktLog *logger.BucktLogger) (domain.CacheManager, domain.LRUCache) {
	fileConf := conf.FileCacheConfig
	fileConf.Validate()

	lruCache, err := cache.NewFileCache(fileConf.NumCounters, fileConf.MaxCost, fileConf.BufferItems)
	if err != nil {
		bucktLog.WrapErrorf("failed to initialize file cache", err)
		lruCache = cache.NewNoOpFileCache()
	}
	bucktLog.Info("✅ Initialized file cache")

	if conf.Manager != nil {
		bucktLog.Info("✅ Using provided cache")
		return conf.Manager, lruCache
	}
	return cache.NewNoOpCache(), lruCache
}

func newAppServices(
	flatNameSpaces bool,
	logger *logger.BucktLogger,
	db *database.DB,
	cacheManager domain.CacheManager,
	activeBackend domain.FileBackend,
) (domain.FolderService, domain.FileService) {
	// Initialize the stores
	var folderRepository domain.FolderRepository = repository.NewFolderRepository(db, logger)
	var fileRepository domain.FileRepository = repository.NewFileRepository(db, logger)

	// initialize the services
	var folderService domain.FolderService = service.NewFolderService(logger, cacheManager, folderRepository, activeBackend)
	var fileService domain.FileService = service.NewFileService(flatNameSpaces, logger, cacheManager, fileRepository, folderService, activeBackend)

	logger.Info("✅ Initialized app services")

	return folderService, fileService
}
