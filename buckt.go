package buckt

import (
	"fmt"
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

	fileService   domain.FileService
	folderService domain.FolderService
}

func New(bucktOpts BucktConfig) (*Buckt, error) {
	buckt := &Buckt{}

	bucktLog := logger.NewLogger(bucktOpts.Log.LogFile, bucktOpts.Log.LogTerminal, logger.WithLogger(bucktOpts.Log.Logger))

	bucktLog.Info("🚀 Starting Buckt")

	// Initialize database
	db, err := database.NewDB(bucktOpts.DB.Database, string(bucktOpts.DB.Driver), bucktLog, bucktOpts.Log.Debug)
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

	// Initialize the stores
	var folderRepository domain.FolderRepository = repository.NewFolderRepository(db, bucktLog)
	var fileRepository domain.FileRepository = repository.NewFileRepository(db, bucktLog)

	// initlize the services
	var folderService domain.FolderService = service.NewFolderService(bucktLog, cacheManager, folderRepository)
	var fileSystemService domain.FileSystemService = service.NewFileSystemService(bucktLog, bucktOpts.MediaDir)
	var fileService domain.FileService = service.NewFileService(bucktLog, cacheManager, bucktOpts.FlatNameSpaces, fileRepository, folderService, fileSystemService)

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

	buckt.db = db
	buckt.router = router
	buckt.fileService = fileService
	buckt.folderService = folderService

	return buckt, nil
}

func Default(opts ...ConfigFunc) (*Buckt, error) {
	bucktOpts := BucktConfig{
		Log:            LogConfig{LogTerminal: true, LogFile: "logs", Debug: true},
		MediaDir:       "media",
		StandaloneMode: true,
		FlatNameSpaces: false,
	}

	for _, opt := range opts {
		opt(&bucktOpts)
	}

	return New(bucktOpts)
}

func (b *Buckt) GetHandler() http.Handler {
	return b.router.Engine
}

func (b *Buckt) StartServer(port string) error {
	return b.router.Run(port)
}

func (b *Buckt) Close() {
	b.db.Close()
}

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

// MoveFile updates the file with the given file_id for the specified user_id.
// It changes the file's name to new_file_name and updates its data to new_file_data.
// Returns an error if the update operation fails.
//
// Parameters:
//   - user_id: The ID of the user who owns the file.
//   - file_id: The ID of the file to be updated.
//   - new_file_name: The new name for the file.
//   - new_file_data: The new data for the file.
//
// Returns:
//   - error: An error if the update operation fails, otherwise nil.
func (b *Buckt) MoveFile(user_id, file_id string, new_file_name string, new_file_data []byte) error {
	return b.fileService.UpdateFile(user_id, file_id, new_file_name, new_file_data)
}

// DeleteFile deletes a file associated with the given user ID and file ID.
// It returns an error if the deletion fails.
//
// Parameters:
//   - user_id: The ID of the user who owns the file.
//   - file_id: The ID of the file to be deleted.
//
// Returns:
//   - error: An error if the file deletion fails, otherwise nil.
func (b *Buckt) DeleteFile(user_id, file_id string) error {
	_, err := b.fileService.DeleteFile(user_id, file_id)
	return err
}

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

// RenameFolder implements Buckt.
func (b *Buckt) RenameFolder(user_id, folder_id string, new_name string) error {
	panic("unimplemented")
}

// DeleteFolder implements Buckt.
func (b *Buckt) DeleteFolder(user_id, folder_id string) error {
	panic("unimplemented")
}
