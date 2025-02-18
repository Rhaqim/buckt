package buckt

import (
	"fmt"
	"mime/multipart"
	"net/http"

	"github.com/Rhaqim/buckt/internal/app"
	"github.com/Rhaqim/buckt/internal/database"
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/repository"
	"github.com/Rhaqim/buckt/internal/service"
	"github.com/Rhaqim/buckt/internal/web/middleware"
	"github.com/Rhaqim/buckt/internal/web/router"
	"github.com/Rhaqim/buckt/pkg/logger"
	"github.com/Rhaqim/buckt/pkg/request"
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
	UploadFile(file *multipart.FileHeader, bucketName string, folderPath string) error
	DownloadFile(req request.FileRequest) ([]byte, error)
	DeleteFile(req request.FileRequest) error
	Serve(filepath string) (string, error)
}

type buckt struct {
	db     *database.DB
	router *router.Router
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
	db, err := database.NewSQLite(log)
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
	var fileService domain.FileService = service.NewFileService(log, fileRepository, folderService, fileSystemService)

	// Initialize the app services
	var apiService domain.APIService = app.NewAPIService(folderService, fileService)
	var webService domain.WebService = app.NewWebService(folderService, fileService)

	// middleware server
	var middleware domain.Middleware = middleware.NewBucketMiddleware()

	// Run the router
	router := router.NewRouter(
		log, tmpl,
		opts.StandaloneMode,
		apiService, webService, middleware)

	return &buckt{
		db,
		router,
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

// DeleteFile implements Buckt.
func (b *buckt) DeleteFile(req request.FileRequest) error {
	panic("unimplemented")
}

// DownloadFile implements Buckt.
func (b *buckt) DownloadFile(req request.FileRequest) ([]byte, error) {
	panic("unimplemented")
}

// Serve implements Buckt.
func (b *buckt) Serve(filepath string) (string, error) {
	panic("unimplemented")
}

// UploadFile implements Buckt.
func (b *buckt) UploadFile(file *multipart.FileHeader, bucketName string, folderPath string) error {
	panic("unimplemented")
}
