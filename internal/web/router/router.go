package router

import (
	"net/http"
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
	httpService domain.APIHTTPService
	middleware.BucketMiddleware
}

// NewRouter creates a new router with the given logger and config.
func NewRouter(log *logger.Logger, cfg *config.Config, httpService domain.APIHTTPService, middleware middleware.BucketMiddleware) *Router {
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
	// middleware := middleware.ClientTypeMiddleware()
	r.Use(middleware.ClientTypeMiddleware())

	router := &Router{r, log, cfg, httpService, middleware}
	router.registerRoutes()

	return router
}

// Run starts the router.
func (r *Router) registerRoutes() {

	// Route for the main admin page (view files)
	r.GET("/portal", func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})

	r.POST("/new_user", r.httpService.NewUser)

	buckets := r.Group("buckets")
	buckets.Use(r.AuthMiddleware())
	{
		buckets.POST("/new_bucket", r.httpService.NewBucket)
		buckets.GET("/fetch", r.httpService.AllBuckets)
		buckets.GET("/fetch/bucket", r.httpService.ViewBucket)
		buckets.DELETE("/delete", r.httpService.RemoveBucket)
	}

	folders := r.Group("folders")
	{
		folders.POST("/fetch/folders", r.httpService.FolderSubFolders)
		folders.POST("/fetch/files", r.httpService.FolderFiles)
		folders.PUT("/rename", r.httpService.FolderRename)
		folders.PUT("/move", r.httpService.FolderMove)
		folders.DELETE("/delete", r.httpService.FolderDelete)

	}

	files := r.Group("files")
	{
		files.POST("/upload", r.httpService.FileUpload)
		files.GET("/download", r.httpService.FileDownload)
		files.PUT("/rename", r.httpService.FileRename)
		files.PUT("/move", r.httpService.FileMove)
		files.GET("/serve", r.httpService.FileServe)
		files.DELETE("/delete", r.httpService.FileDelete)
	}

}

// ServeHTTP makes Router compatible with http.Handler
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.Engine.ServeHTTP(w, req)
}
