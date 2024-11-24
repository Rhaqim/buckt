package router

import (
	"path/filepath"
	"runtime"

	"github.com/Rhaqim/buckt/config"
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/web/middleware"
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

	// Set the client type middleware i.e. portal or api
	middleware := middleware.ClientTypeMiddleware()
	r.Use(middleware)

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
	r.GET("/api/fetch", r.httpService.FetchFilesInFolder)
	r.GET("/api/fetch/folders", r.httpService.FetchSubFolders)

	// folders := r.Group("folders")
	// {
	// 	folders.GET("/fetch", r.httpService.FetchSubFolders)
	// 	folders.GET("/fetch/files", r.httpService.FetchFilesInFolder)
	// 	folders.PUT("/rename", r.httpService.RenameFolder)
	// 	folders.PUT("/move", r.httpService.MoveFolder)
	// 	folders.DELETE("/delete", r.httpService.DeleteFolder)

	// }

	// files := r.Group("files")
	// {
	// 	files.PUT("/rename", r.httpService.RenameFile)
	// 	files.PUT("/move", r.httpService.MoveFile)
	// 	files.DELETE("/delete", r.httpService.DeleteFile)
	// 	files.GET("/download", r.httpService.Download)
	// 	files.GET("/serve", r.httpService.ServeFile)
	// }

	return r.Engine.Run(r.Server.Port)
}

// Define the routes for the router
