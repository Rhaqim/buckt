package router

import (
	"net/http"
	"path/filepath"
	"runtime"

	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/pkg/logger"
	"github.com/gin-gonic/gin"
)

type Router struct {
	*gin.Engine

	domain.APIService
	domain.WebService
	domain.Middleware
}

// NewRouter creates a new router with the given logger and config.
func NewRouter(
	log *logger.Logger,
	templateDir string,
	apiService domain.APIService,
	webService domain.WebService,
	middleware domain.Middleware,
) *Router {
	r := gin.New()

	// Set logger
	r.Use(gin.LoggerWithWriter(log.Writer()))

	// Set recovery
	r.Use(gin.Recovery())

	// Determine base path for templates
	_, b, _, _ := runtime.Caller(0)
	basePath := filepath.Dir(b)
	templatePath := templateDir

	// If no specific template path is set, use the default pattern
	if templatePath == "" {
		templatePath = filepath.Join(basePath, "..", "templates", "*.html")
	} else {
		// Ensure the provided path has a wildcard pattern
		templatePath = filepath.Join(templatePath, "*.html")
	}

	// Load templates using the specified pattern
	r.LoadHTMLGlob(templatePath)

	router := &Router{
		Engine: r,

		APIService: apiService,
		WebService: webService,
		Middleware: middleware,
	}
	router.registerRoutes()

	return router
}

// Run starts the router.
func (r *Router) registerRoutes() {
	r.GET("/serve/:file_id", r.APIService.ServeFile)

	/* Web Routes */
	r.Use(r.WebGuardMiddleware())
	{
		r.GET("/", r.WebService.ViewFolder)
		r.GET("/folder/:folder_id", r.WebService.ViewFolder)
	}

	/* API Routes */
	api := r.Group("/api")

	api.Use(r.APIGuardMiddleware())
	{
		{
			api.POST("/upload", r.APIService.UploadFile)
			api.GET("/download", r.APIService.DownloadFile)
			api.DELETE("/delete", r.APIService.DeleteFile)
		}

		{
			api.POST("/new_folder", r.APIService.CreateFolder)
			api.GET("/folder_content", r.APIService.GetFolderContent)
			api.GET("/folder_folders", r.APIService.GetSubFolders)
			api.GET("/folder_files", r.APIService.GetFilesInFolder)
			api.GET("/folder_descendants", r.APIService.GetDescendants)
			api.PUT("/rename_folder", r.APIService.RenameFolder)
			api.PUT("/move_folder", r.APIService.MoveFolder)
			api.DELETE("/delete_folder", r.APIService.DeleteFolder)
		}
	}

	// r.POST("/new_user", r.APIService.NewUser)

	// buckets := r.Group("buckets")
	// buckets.Use(r.AuthMiddleware())
	// {
	// 	buckets.POST("/new_bucket", r.APIService.NewBucket)
	// 	buckets.GET("/fetch", r.APIService.AllBuckets)
	// 	buckets.GET("/fetch/bucket", r.APIService.ViewBucket)
	// 	buckets.DELETE("/delete", r.APIService.RemoveBucket)
	// }

	// folders := r.Group("folders/:bucket")
	// {
	// 	folders.GET("", r.APIService.FolderContent)
	// 	folders.GET("/fetch/folders", r.APIService.FolderSubFolders)
	// 	folders.GET("/fetch/files", r.APIService.FolderFiles)
	// 	folders.PUT("/rename", r.APIService.FolderRename)
	// 	folders.PUT("/move", r.APIService.FolderMove)
	// 	folders.DELETE("/delete", r.APIService.FolderDelete)

	// }

	// files := r.Group("files/:bucket")
	// {
	// 	files.POST("/upload", r.APIService.FileUpload)
	// 	files.GET("/download", r.APIService.FileDownload)
	// 	files.PUT("/rename", r.APIService.FileRename)
	// 	files.PUT("/move", r.APIService.FileMove)
	// 	files.GET("/serve", r.APIService.FileServe)
	// 	files.DELETE("/delete", r.APIService.FileDelete)
	// }

}

// ServeHTTP makes Router compatible with http.Handler
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.Engine.ServeHTTP(w, req)
}
