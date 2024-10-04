package router

import (
	"github.com/Rhaqim/buckt/config"
	"github.com/Rhaqim/buckt/internal/database"
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/Rhaqim/buckt/internal/service"
	"github.com/Rhaqim/buckt/pkg/logger"
	"github.com/gin-gonic/gin"
)

type Router struct {
	*gin.Engine
	*logger.Logger
	*config.Config
	*database.DB
}

// NewRouter creates a new router with the given logger and config.
func NewRouter(log *logger.Logger, cfg *config.Config, db *database.DB) *Router {
	r := gin.New()

	// Set logger
	r.Use(gin.LoggerWithWriter(log.InfoLogger.Writer()))

	// Set recovery
	r.Use(gin.Recovery())

	return &Router{r, log, cfg, db}
}

// Run starts the router.
func (r *Router) Run() error {

	var fileStore domain.Repository[model.FileModel] = model.NewFileRepository(r.DB.DB)

	var bucketStore domain.Repository[model.BucketModel] = model.NewBucketRepository(r.DB.DB)

	var ownerStore domain.Repository[model.OwnerModel] = model.NewOwnerRepository(r.DB.DB)

	var tagStore domain.Repository[model.TagModel] = model.NewTagRepository(r.DB.DB)

	var fileService domain.StorageFileService = service.NewStorageService(r.Logger, r.Config, fileStore, bucketStore, ownerStore, tagStore)

	var httpService domain.StorageHTTPService = service.NewHTTPService(fileService)

	r.POST("/new_user", httpService.NewUser)
	r.POST("/new_bucket", httpService.NewBucket)

	r.POST("/upload", httpService.Upload)
	r.GET("/download/:filename", httpService.Download)
	r.GET("/serve/:filename", httpService.ServeFile)
	r.DELETE("/delete/:filename", httpService.Delete)

	return r.Engine.Run(r.Server.Port)
}
