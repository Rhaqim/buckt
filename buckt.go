package buckt

import (
	"github.com/Rhaqim/buckt/config"
	"github.com/Rhaqim/buckt/internal/database"
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/Rhaqim/buckt/internal/service"
	"github.com/Rhaqim/buckt/internal/web/router"
	"github.com/Rhaqim/buckt/pkg/logger"
)

// Buckt is the interface for the Buckt service
type Buckt interface {
	// Buckt HTTP service methods
	Start() error
	Close()

	// Buckt storage service methods
	UploadFile(file []byte, bucketname, filename string) error
	DownloadFile(filename string) ([]byte, error)
	DeleteFile(filename string) error
	CreateBucket(name, description, ownerID string) error
	CreateOwner(name, email string) error
	Serve(filename string, serve bool) (string, error)
}

type buckt struct {
	cfg *config.Config
	log *logger.Logger
	db  *database.DB
	domain.StorageFileService
	router *router.Router
}

func NewBuckt(configFile string, logToFileAndTerminal bool, saveDir string) (Buckt, error) {
	// Load config
	cfg := config.LoadConfig(configFile)

	// Initialize logger
	log := logger.NewLogger(false, logToFileAndTerminal, saveDir)

	// Initialize database
	db, err := database.NewSQLite(cfg, log)
	if err != nil {
		return nil, err
	}

	// Migrate the database
	db.Migrate()

	// Initialize the stores
	var fileStore domain.Repository[model.FileModel] = model.NewFileRepository(db.DB)
	var bucketStore domain.Repository[model.BucketModel] = model.NewBucketRepository(db.DB)
	var ownerStore domain.Repository[model.OwnerModel] = model.NewOwnerRepository(db.DB)
	var tagStore domain.Repository[model.TagModel] = model.NewTagRepository(db.DB)

	// Initialize the services
	var fileService domain.StorageFileService = service.NewStorageService(log, cfg, fileStore, bucketStore, ownerStore, tagStore)

	// Http service
	var httpService domain.StorageHTTPService = service.NewHTTPService(fileService)

	// Run the router
	router := router.NewRouter(log, cfg, httpService)

	return &buckt{
		cfg,
		log,
		db,
		fileService,
		router,
	}, nil
}

func (b *buckt) Start() error {
	return b.router.Run()
}

func (b *buckt) Close() {
	b.db.Close()
}

func (b *buckt) UploadFile(file []byte, bucketname, filename string) error {
	return b.StorageFileService.UploadFile(file, bucketname, filename)
}

func (b *buckt) DownloadFile(filename string) ([]byte, error) {
	return b.StorageFileService.DownloadFile(filename)
}

func (b *buckt) DeleteFile(filename string) error {
	return b.StorageFileService.DeleteFile(filename)
}

func (b *buckt) CreateBucket(name, description, ownerID string) error {
	return b.StorageFileService.CreateBucket(name, description, ownerID)
}

func (b *buckt) CreateOwner(name, email string) error {
	return b.StorageFileService.CreateOwner(name, email)
}

func (b *buckt) Serve(filename string, serve bool) (string, error) {
	return b.StorageFileService.Serve(filename, serve)
}
