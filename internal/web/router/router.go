package router

import (
	"html/template"
	"net/http"

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
	tmpl *template.Template,

	StandaloneMode bool,

	apiService domain.APIService,
	webService domain.WebService,
	middleware domain.Middleware,
) *Router {
	r := gin.New()

	// Set logger
	r.Use(gin.LoggerWithWriter(log.Writer()))

	// Set recovery
	r.Use(gin.Recovery())

	// Set HTML template
	r.SetHTMLTemplate(tmpl)

	router := &Router{
		Engine: r,

		APIService: apiService,
		WebService: webService,
		Middleware: middleware,
	}

	router.registerAllRoutes(StandaloneMode)

	return router
}

// RegisterAllRoutes registers all routes for the router.
func (r *Router) RegisterBaseRoutes() {
	r.GET("/serve/:file_id", r.APIService.ServeFile)
}

// RegisterAPIRoutes sets up API endpoints
func (r *Router) RegisterAPIRoutes() {
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
}

// RegisterWebRoutes sets up the web interface routes
func (r *Router) RegisterWebRoutes() {
	/* Web Routes */
	r.Use(r.WebGuardMiddleware())
	{
		r.GET("/", r.WebService.ViewFolder)
		r.GET("/folder/:folder_id", r.WebService.ViewFolder)
		r.POST("/new-folder", r.WebService.NewFolder)
		r.PUT("/rename-folder", r.WebService.RenameFolder)
		r.PUT("/move-folder", r.WebService.MoveFolder)
		r.DELETE("/folder/:folder_id", r.WebService.DeleteFolder)

		r.POST("/upload", r.WebService.UploadFile)
		r.GET("/file/:file_id", r.WebService.DownloadFile)
		r.PUT("/file/:file_id", r.WebService.MoveFile)
		r.DELETE("/file/:file_id", r.WebService.DeleteFile)
	}
}

// registerAllRoutes registers all required routes
func (r *Router) registerAllRoutes(StandaloneMode bool) {
	// Register core routes
	r.RegisterBaseRoutes()

	// Register API routes
	r.RegisterAPIRoutes()

	// Register web routes **only if in standalone mode**
	if StandaloneMode {
		r.RegisterWebRoutes()
	}
}

// ServeHTTP makes Router compatible with http.Handler
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.Engine.ServeHTTP(w, req)
}
