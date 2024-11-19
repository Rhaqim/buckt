package router

import (
	"path/filepath"
	"runtime"

	"github.com/Rhaqim/buckt/config"
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/pkg/logger"
	"github.com/gin-gonic/gin"
)

type Router struct {
	*gin.Engine
	*logger.Logger
	*config.Config
	httpService   domain.StorageHTTPService
	portalService domain.StorageHTTPService
}

// NewRouter creates a new router with the given logger and config.
func NewRouter(log *logger.Logger, cfg *config.Config, httpService, portalService domain.StorageHTTPService) *Router {
	r := gin.New()

	// Set logger
	r.Use(gin.LoggerWithWriter(log.InfoLogger.Writer()))

	// Set recovery
	r.Use(gin.Recovery())

	// Determine base path for templates
	_, b, _, _ := runtime.Caller(0)
	basePath := filepath.Dir(b)
	templatePath := cfg.Templates

	// If no specific template path is set, use the default pattern
	if templatePath == "" {
		templatePath = filepath.Join(basePath, "..", "templates", "*.html")
	} else {
		// Ensure the provided path has a wildcard pattern
		templatePath = filepath.Join(templatePath, "*.html")
	}

	// Load templates using the specified pattern
	r.LoadHTMLGlob(templatePath)

	return &Router{r, log, cfg, httpService, portalService}
}

// Run starts the router.
func (r *Router) Run() error {

	// Route for the main admin page (view files)
	r.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})

	// Route for creating a new bucket
	r.POST("/new_bucket", r.portalService.NewBucket)

	// Route for uploading a new file
	r.POST("/upload", r.portalService.Upload)

	// Route for downloading an existing file
	r.GET("/download/:filename", r.portalService.Download)

	// Route for deleting an existing file
	r.DELETE("/delete/:filename", r.portalService.Delete)

	// Route for viewing files in a specific bucket
	r.GET("/view/:bucket_name", r.portalService.FetchFiles)

	r.POST("/api/new_user", r.httpService.NewUser)
	r.POST("/api/new_bucket", r.httpService.NewBucket)
	r.POST("/api/upload", r.httpService.Upload)
	r.GET("/api/download/:filename", r.httpService.Download)
	r.DELETE("/api/delete/:filename", r.httpService.Delete)
	r.GET("/api/serve/:filename", r.httpService.ServeFile)

	return r.Engine.Run(r.Server.Port)
}
