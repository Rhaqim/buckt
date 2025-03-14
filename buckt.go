package buckt

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/Rhaqim/buckt/internal/app"
	"github.com/Rhaqim/buckt/internal/cache"
	"github.com/Rhaqim/buckt/internal/database"
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/Rhaqim/buckt/internal/repository"
	"github.com/Rhaqim/buckt/internal/service"
	"github.com/Rhaqim/buckt/internal/web/middleware"
	"github.com/Rhaqim/buckt/internal/web/router"
	"github.com/Rhaqim/buckt/pkg/logger"
)

type Buckt struct {
	db     *database.DB
	router *router.Router

	flatnameSpaces bool

	fileService   domain.FileService
	folderService domain.FolderService

	cloudService domain.CloudService
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
func New(bucktOpts BucktConfig) (*Buckt, error) {
	buckt := &Buckt{}

	bucktLog := logger.NewLogger(bucktOpts.Log.LogFile, bucktOpts.Log.LogTerminal, bucktOpts.Log.Debug, logger.WithLogger(bucktOpts.Log.Logger))

	bucktLog.Info("🚀 Starting Buckt")

	// Initialize database
	db, err := database.NewDB(bucktOpts.DB.Database, bucktOpts.DB.Driver, bucktLog, bucktOpts.Log.Debug)
	if err != nil {
		return nil, err
	}

	// Migrate the database
	err = db.Migrate()
	if err != nil {
		return nil, err
	}

	// Cache
	var cacheManager domain.CacheManager
	if bucktOpts.Cache != nil {
		bucktLog.Info("🚀 Using provided cache")
		cacheManager = bucktOpts.Cache
	} else {
		cacheManager = cache.NewNoOpCache()
	}

	// Load templates
	bucktLog.Info("🚀 Loading templates")
	tmpl, err := loadTemplates()
	if err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	// Initialize the app services
	folderService, fileService := newAppServices(bucktLog, bucktOpts, db, cacheManager)

	// Initialize the router
	router := newRouterService(tmpl, bucktLog, bucktOpts, fileService, folderService)

	buckt.db = db
	buckt.router = router
	buckt.flatnameSpaces = bucktOpts.FlatNameSpaces
	buckt.fileService = fileService
	buckt.folderService = folderService

	// Initialize cloud service if provided
	if !bucktOpts.Cloud.isEmpty() {
		fmt.Println("🚀 Initializing cloud service")
		err = buckt.InitCloudService(bucktOpts.Cloud)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize cloud service: %w", err)
		}
	}

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
func Default(opts ...ConfigFunc) (*Buckt, error) {
	bucktOpts := BucktConfig{
		Log:            LogConfig{LogTerminal: true, Debug: true},
		MediaDir:       "media",
		StandaloneMode: true,
		FlatNameSpaces: true,
	}

	for _, opt := range opts {
		opt(&bucktOpts)
	}

	return New(bucktOpts)
}

// GetHandler returns the HTTP handler for the Buckt instance.
// It provides access to the underlying router engine.
func (b *Buckt) GetHandler() http.Handler {
	return b.router.Handler()
}

// StartServer starts the server on the specified port using the router.
// It takes a port string as an argument and returns an error if the server fails to start.
//
// Parameters:
//
//	port (string): The port on which the server will listen.
//
// Returns:
//
//	error: An error if the server fails to start, otherwise nil.
func (b *Buckt) StartServer(port string) error {
	return b.router.Run(port)
}

// Close closes the Buckt instance.
// It closes the database connection.
func (b *Buckt) Close() {
	b.db.Close()
}

/* Folder Methods */

// NewFolder creates a new folder for a user within a specified parent folder.
// It takes the following parameters:
// - user_id: The ID of the user creating the folder.
// - parent_id: The ID of the parent folder where the new folder will be created.
// - folder_name: The name of the new folder.
// - description: A description of the new folder.
// It returns the ID of the newly created folder and an error if the operation fails.
func (b *Buckt) NewFolder(user_id string, parent_id string, folder_name string, description string) (new_folder_id string, err error) {
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
func (b *Buckt) ListFolders(folder_id string) ([]model.FolderModel, error) {
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
func (b *Buckt) GetFolderWithContent(user_id, folder_id string) (*model.FolderModel, error) {
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
func (b *Buckt) MoveFolder(user_id, folder_id string, new_parent_id string) error {
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
func (b *Buckt) RenameFolder(user_id, folder_id string, new_name string) error {
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
func (b *Buckt) DeleteFolder(folder_id string) error {
	_, err := b.folderService.DeleteFolder(folder_id)
	return err
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
func (b *Buckt) DeleteFolderPermanently(user_id, folder_id string) error {

	// If flatnameSpaces is enabled, we soft delete the folder
	if b.flatnameSpaces {
		_, err := b.folderService.DeleteFolder(folder_id)

		return err
	}

	_, err := b.folderService.ScrubFolder(user_id, folder_id)

	return err
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
func (b *Buckt) UploadFile(user_id string, parent_id string, file_name string, content_type string, file_data []byte) (string, error) {
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
func (b *Buckt) GetFile(file_id string) (*model.FileModel, error) {
	return b.fileService.GetFile(file_id)
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
func (b *Buckt) ListFiles(folder_id string) ([]model.FileModel, error) {
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
func (b *Buckt) MoveFile(file_id string, new_parent_id string) error {
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
func (b *Buckt) DeleteFile(file_id string) error {
	_, err := b.fileService.DeleteFile(file_id)
	return err
}

// DeleteFilePermanently deletes a file associated with the given user ID and file ID.
// It returns an error if the deletion fails.
//
// Parameters:
//   - file_id: The ID of the file to be deleted.
//
// Returns:
//   - error: An error if the file deletion fails, otherwise nil.
func (b *Buckt) DeleteFilePermanently(file_id string) error {
	_, err := b.fileService.ScrubFile(file_id)
	return err
}

/* Cloud Methods */

// InitCloudService initializes the cloud service with the provided cloud configuration.
// It takes a CloudConfig struct as an argument and returns an error if the initialization fails.
//
// Parameters:
//   - cloudConfig: A CloudConfig struct containing the configuration options for the cloud service.
//
// Returns:
//   - error: An error if the cloud service initialization fails, otherwise nil.
func (b *Buckt) InitCloudService(cloudConfig CloudConfig) error {
	var err error

	if cloudConfig.isEmpty() {
		return fmt.Errorf("cloud configuration is empty")
	}

	fmt.Println("🚀 Initializing cloud service")

	// Initialize the cloud service
	b.cloudService, err = initCloudClient(cloudConfig, b.fileService, b.folderService)

	return err
}

// TransferFile transfers a file from the local storage to the cloud storage.
// It takes the file ID as a parameter and returns an error if the transfer fails.
//
// Parameters:
//   - file_id: The ID of the file to be transferred.
//
// Returns:
//   - error: An error if the transfer fails, otherwise nil.
func (b *Buckt) TransferFile(file_id string) error {
	if b.cloudService == nil {
		return fmt.Errorf("cloud service not initialized")
	}

	return b.cloudService.UploadFileToCloud(file_id)
}

// TransferFolder transfers a folder from the local storage to the cloud storage.
// It takes the user_id and folder ID as a parameter and returns an error if the transfer fails.
//
// Parameters:
//   - user_id: The ID of the user who owns the folder.
//   - folder_id: The ID of the folder to be transferred.
//
// Returns:
//   - error: An error if the transfer fails, otherwise nil.
func (b *Buckt) TransferFolder(user_id, folder_id string) error {
	if b.cloudService == nil {
		return fmt.Errorf("cloud service not initialized")
	}

	return b.cloudService.UploadFolderToCloud(user_id, folder_id)
}

/* Helper Methods */

func newAppServices(bucktLog *logger.BucktLogger, bucktOpts BucktConfig, db *database.DB, cacheManager domain.CacheManager) (domain.FolderService, domain.FileService) {
	// Initialize the stores
	var folderRepository domain.FolderRepository = repository.NewFolderRepository(db, bucktLog)
	var fileRepository domain.FileRepository = repository.NewFileRepository(db, bucktLog)

	// initlize the services
	var fileSystemService domain.FileSystemService = service.NewFileSystemService(bucktLog, bucktOpts.MediaDir)
	var folderService domain.FolderService = service.NewFolderService(bucktLog, cacheManager, folderRepository, fileSystemService)
	var fileService domain.FileService = service.NewFileService(bucktLog, cacheManager, bucktOpts.FlatNameSpaces, fileRepository, folderService, fileSystemService)

	return folderService, fileService
}

func newRouterService(tmpl *template.Template, bucktLog *logger.BucktLogger, bucktOpts BucktConfig, fileService domain.FileService, folderService domain.FolderService) *router.Router {
	// Initialize the app services
	var apiService domain.APIService = app.NewAPIService(folderService, fileService)
	var webService domain.WebService = app.NewWebService(folderService, fileService)

	// middleware server
	var middleware domain.Middleware = middleware.NewBucketMiddleware(bucktLog, bucktOpts.StandaloneMode)

	// Run the router
	router := router.NewRouter(
		bucktLog, tmpl,
		bucktOpts.Log.Debug,
		bucktOpts.StandaloneMode,
		apiService, webService, middleware)

	return router
}
