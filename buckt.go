package buckt

import (
	"fmt"
	"net/http"

	"github.com/Rhaqim/buckt/internal/app"
	"github.com/Rhaqim/buckt/internal/database"
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/repository"
	"github.com/Rhaqim/buckt/internal/service"
	"github.com/Rhaqim/buckt/internal/web/middleware"
	"github.com/Rhaqim/buckt/internal/web/router"
	"github.com/Rhaqim/buckt/pkg/logger"
)

// Buckt is the interface for the Buckt service
type Buckt interface {
	// GetHandler returns the http.Handler for the Buckt service.
	GetHandler() http.Handler

	// StartServer starts the Buckt service on the specified port.
	StartServer(port ...string) error

	// Close closes the Buckt service.
	Close()

	// Buckt service methods
	UploadFile(user_id string, parent_id string, file_name string, content_type string, file_data []byte) (string, error)
	GetFile(file_id string) (interface{}, error)
	MoveFile(file_id string, new_file_name string, new_file_data []byte) error
	DeleteFile(file_id string) error
	NewFolder(user_id string, parent_id string, folder_name string, description string) error
	GetFolder(user_id string, folder_id string) (interface{}, error)
	GetFolderContent(parent_id string) ([]interface{}, error)
	MoveFolder(folder_id string, new_parent_id string) error
	RenameFolder(folder_id string, new_name string) error
	DeleteFolder(folder_id string) error
}

type buckt struct {
	db            *database.DB
	router        *router.Router
	fileService   domain.FileService
	folderService domain.FolderService
}

// NewBuckt initializes and returns a new Buckt instance.
// It sets up the logger, database, templates, repositories, services, and router.
//
// Parameters:
//   - opts: BucktOptions containing configuration options for the Buckt instance.
//
// Returns:
//   - Buckt: The initialized Buckt instance.
//   - error: An error if any step in the initialization process fails.
func NewBuckt(opts BucktOptions) (Buckt, error) {
	// Initialize logger
	log := logger.NewLogger(opts.Log.LoGfILE, opts.Log.LogTerminal)

	// Initialize database
	db, err := database.NewSQLite(log, opts.Log.Debug)
	if err != nil {
		return nil, err
	}

	// Migrate the database
	err = db.Migrate()
	if err != nil {
		return nil, err
	}

	// Load templates
	tmpl, err := loadTemplates()
	if err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	// Initialize the stores
	var folderRepository domain.FolderRepository = repository.NewFolderRepository(db, log)
	var fileRepository domain.FileRepository = repository.NewFileRepository(db, log)

	// initlize the services
	var folderService domain.FolderService = service.NewFolderService(log, folderRepository)
	var fileSystemService domain.FileSystemService = service.NewFileSystemService(log, opts.MediaDir)
	var fileService domain.FileService = service.NewFileService(log, opts.FlatNameSpaces, fileRepository, folderService, fileSystemService)

	// Initialize the app services
	var apiService domain.APIService = app.NewAPIService(folderService, fileService)
	var webService domain.WebService = app.NewWebService(folderService, fileService)

	// middleware server
	var middleware domain.Middleware = middleware.NewBucketMiddleware()

	// Run the router
	router := router.NewRouter(
		log, tmpl,
		opts.Log.Debug,
		opts.StandaloneMode,
		apiService, webService, middleware)

	return &buckt{
		db:            db,
		router:        router,
		fileService:   fileService,
		folderService: folderService,
	}, nil
}

func (b *buckt) GetHandler() http.Handler {
	return b.router.Engine
}

func (b *buckt) StartServer(port ...string) error {
	return b.router.Engine.Run(port...)
}

func (b *buckt) Close() {
	b.db.Close()
}

// CreateFile implements Buckt.
func (b *buckt) UploadFile(user_id string, parent_id string, file_name string, content_type string, file_data []byte) (string, error) {
	return b.fileService.CreateFile(user_id, parent_id, file_name, content_type, file_data)
}

// GetFile implements Buckt.
func (b *buckt) GetFile(file_id string) (interface{}, error) {
	return b.fileService.GetFile(file_id)
}

// UpdateFile implements Buckt.
func (b *buckt) MoveFile(file_id string, new_file_name string, new_file_data []byte) error {
	return b.fileService.UpdateFile(file_id, new_file_name, new_file_data)
}

// DeleteFile implements Buckt.
func (b *buckt) DeleteFile(file_id string) error {
	return b.fileService.DeleteFile(file_id)
}

// CreateFolder implements Buckt.
func (b *buckt) NewFolder(user_id string, parent_id string, folder_name string, description string) error {
	return b.folderService.CreateFolder(user_id, parent_id, folder_name, description)
}

// GetFolder implements Buckt.
func (b *buckt) GetFolder(user_id string, folder_id string) (interface{}, error) {
	return b.folderService.GetFolder(user_id, folder_id)
}

// GetFolders implements Buckt.
func (b *buckt) GetFolderContent(parent_id string) ([]interface{}, error) {
	folders, err := b.folderService.GetFolders(parent_id)
	if err != nil {
		return nil, err
	}

	files, err := b.fileService.GetFiles(parent_id)
	if err != nil {
		return nil, err
	}

	content := make([]interface{}, 0, len(folders)+len(files))
	for _, folder := range folders {
		content = append(content, folder)
	}
	for _, file := range files {
		content = append(content, file)
	}

	return content, nil
}

// MoveFolder implements Buckt.
func (b *buckt) MoveFolder(folder_id string, new_parent_id string) error {
	panic("unimplemented")
}

// RenameFolder implements Buckt.
func (b *buckt) RenameFolder(folder_id string, new_name string) error {
	panic("unimplemented")
}

// DeleteFolder implements Buckt.
func (b *buckt) DeleteFolder(folder_id string) error {
	panic("unimplemented")
}
