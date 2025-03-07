package buckt

import (
	"fmt"
	"net/http"

	"github.com/Rhaqim/buckt/internal/app"
	"github.com/Rhaqim/buckt/internal/cache"
	"github.com/Rhaqim/buckt/internal/database"
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/repository"
	"github.com/Rhaqim/buckt/internal/service"
	"github.com/Rhaqim/buckt/internal/web/middleware"
	"github.com/Rhaqim/buckt/internal/web/router"
	"github.com/Rhaqim/buckt/pkg/logger"
)

type Buckt struct {
	DB     *database.DB
	router *router.Router

	fileService   domain.FileService
	folderService domain.FolderService
}

func New(bucktOpts BucktConfig) (*Buckt, error) {
	buckt := &Buckt{}

	bucktLog := logger.NewLogger(bucktOpts.Log.LoGfILE, bucktOpts.Log.LogTerminal, logger.WithLogger(bucktOpts.Log.Logger))

	bucktLog.Info("🚀 Starting Buckt")

	// Initialize database
	db, err := database.NewDB(bucktOpts.DB, bucktLog, bucktOpts.Log.Debug)
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

	buckt.DB = db
	buckt.router = router
	buckt.fileService = fileService
	buckt.folderService = folderService

	return buckt, nil
}

func Default(opts ...ConfigFunc) (*Buckt, error) {
	bucktOpts := BucktConfig{
		Log:            Log{LogTerminal: true, LoGfILE: "logs", Debug: true},
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
	b.DB.Close()
}

// CreateFile implements Buckt.
func (b *Buckt) UploadFile(user_id string, parent_id string, file_name string, content_type string, file_data []byte) (string, error) {
	return b.fileService.CreateFile(user_id, parent_id, file_name, content_type, file_data)
}

// GetFile implements Buckt.
func (b *Buckt) GetFile(file_id string) (any, error) {
	return b.fileService.GetFile(file_id)
}

// UpdateFile implements Buckt.
func (b *Buckt) MoveFile(user_id, file_id string, new_file_name string, new_file_data []byte) error {
	return b.fileService.UpdateFile(user_id, file_id, new_file_name, new_file_data)
}

// DeleteFile implements Buckt.
func (b *Buckt) DeleteFile(user_id, file_id string) error {
	_, err := b.fileService.DeleteFile(file_id)
	return err
}

// CreateFolder implements Buckt.
func (b *Buckt) NewFolder(user_id string, parent_id string, folder_name string, description string) (new_folder_id string, err error) {
	return b.folderService.CreateFolder(user_id, parent_id, folder_name, description)
}

// GetFolder implements Buckt.
func (b *Buckt) GetFolder(user_id string, folder_id string) (any, error) {
	return b.folderService.GetFolder(user_id, folder_id)
}

// GetFolders implements Buckt.
func (b *Buckt) GetFolderContent(user_id, folder_id string) (any, error) {
	return b.folderService.GetFolder(user_id, folder_id)
}

// MoveFolder implements Buckt.
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
