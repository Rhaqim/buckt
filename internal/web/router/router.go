package router

import (
	"github.com/Rhaqim/buckt/config"
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/pkg/logger"
	"github.com/gin-gonic/gin"
)

type Router struct {
	*gin.Engine
	*logger.Logger
	*config.Config
	httpService domain.StorageHTTPService
}

// NewRouter creates a new router with the given logger and config.
func NewRouter(log *logger.Logger, cfg *config.Config, httpService domain.StorageHTTPService) *Router {
	r := gin.New()

	// Set logger
	r.Use(gin.LoggerWithWriter(log.InfoLogger.Writer()))

	// Set recovery
	r.Use(gin.Recovery())

	return &Router{r, log, cfg, httpService}
}

// Run starts the router.
func (r *Router) Run() error {

	r.POST("/new_user", r.httpService.NewUser)
	r.POST("/new_bucket", r.httpService.NewBucket)

	r.POST("/upload", r.httpService.Upload)
	r.GET("/download/:filename", r.httpService.Download)
	r.DELETE("/delete/:filename", r.httpService.Delete)
	r.GET("/serve/:filename", r.httpService.ServeFile)

	return r.Engine.Run(r.Server.Port)
}
