package buckt

import (
	"mime/multipart"
	"net/http"

	"github.com/Rhaqim/buckt/config"
	"github.com/Rhaqim/buckt/internal/api"
	"github.com/Rhaqim/buckt/internal/database"
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/Rhaqim/buckt/internal/service"
	"github.com/Rhaqim/buckt/internal/web/middleware"
	"github.com/Rhaqim/buckt/internal/web/router"
	"github.com/Rhaqim/buckt/pkg/logger"
	"github.com/Rhaqim/buckt/request"
)

// Buckt is the interface for the Buckt service
type Buckt interface {
	// Buckt HTTP service methods
	GetHandler() http.Handler
	StartServer(port ...string) error
	Close()

	// Buckt storage service methods
	UploadFile(file *multipart.FileHeader, bucketName string, folderPath string) error
	DownloadFile(req request.FileRequest) ([]byte, error)
	DeleteFile(req request.FileRequest) error
	CreateBucket(name, description, ownerID string) error
	CreateOwner(name, email string) error
	Serve(filepath string) (string, error)
}

type buckt struct {
	db *database.DB
	domain.BucktService
	router *router.Router
}

func NewBuckt(configFile string) (Buckt, error) {
	// Load config
	cfg := config.LoadConfig(configFile)

	// Initialize logger
	log := logger.NewLogger(cfg.Log.LoGfILE, cfg.Log.LogTerminal)

	// Initialize database
	db, err := database.NewSQLite(cfg, log)
	if err != nil {
		return nil, err
	}

	// Migrate the database
	db.Migrate()

	// Initialize the stores
	var tagStore domain.BucktRepository[model.TagModel] = model.NewTagRepository(db.DB)
	var fileStore domain.BucktRepository[model.FileModel] = model.NewFileRepository(db.DB)
	var folderStore domain.BucktRepository[model.FolderModel] = model.NewFolderRepository(db.DB)
	var bucketStore domain.BucktRepository[model.BucketModel] = model.NewBucketRepository(db.DB)
	var ownerStore domain.BucktRepository[model.OwnerModel] = model.NewOwnerRepository(db.DB)

	store := &model.BucktStore{
		OwnerStore:  ownerStore,
		BucketStore: bucketStore,
		FolderStore: folderStore,
		FileStore:   fileStore,
		TagStore:    tagStore,
	}

	// Initialize the services
	var fileService domain.BucktService = service.NewBucktService(log, cfg, store)

	// API service
	var httpService domain.APIHTTPService = api.NewAPIService(fileService)

	// middleware server
	middleware := middleware.NewBucketMiddleware(ownerStore)

	// Run the router
	router := router.NewRouter(log, cfg, httpService, middleware)

	return &buckt{
		db,
		fileService,
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

func (b *buckt) UploadFile(file *multipart.FileHeader, bucketName string, folderPath string) error {
	return b.BucktService.UploadFile(file, bucketName, folderPath)
}

func (b *buckt) DownloadFile(req request.FileRequest) ([]byte, error) {
	return b.BucktService.DownloadFile(req)
}

func (b *buckt) DeleteFile(req request.FileRequest) error {
	return b.BucktService.DeleteFile(req)
}

func (b *buckt) CreateBucket(name, description, ownerID string) error {
	return b.BucktService.CreateBucket(name, description, ownerID)
}

func (b *buckt) CreateOwner(name, email string) error {
	return b.BucktService.CreateOwner(name, email)
}

func (b *buckt) Serve(filepath string) (string, error) {
	return b.BucktService.ServeFile(filepath)
}
