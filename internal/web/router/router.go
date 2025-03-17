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

	StandaloneMode bool

	domain.APIService
	domain.WebService
	domain.Middleware
}

// NewRouter creates a new router with the given logger and config.
func NewRouter(
	log *logger.BucktLogger,
	tmpl *template.Template,

	Debug bool,
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

	// Release mode
	if !Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	// Set HTML template
	r.SetHTMLTemplate(tmpl)

	router := &Router{
		Engine: r,

		StandaloneMode: StandaloneMode,

		APIService: apiService,
		WebService: webService,
		Middleware: middleware,
	}

	return router
}

// Run starts the router on the given address.
func (r *Router) Run(addr string) error {
	r.registerAllRoutes(r.StandaloneMode)

	return r.Engine.Run(addr)
}

// ServeHTTP makes Router compatible with http.Handler
func (r *Router) Handler() http.Handler {
	r.registerAllRoutes(r.StandaloneMode)

	return r.Engine
}

// RegisterAllRoutes registers all routes for the router.
func (r *Router) registerBaseRoutes() {
	r.GET("/", func(c *gin.Context) {
		// redirect to /web
		c.Redirect(http.StatusMovedPermanently, "/web")
	})
	r.GET("/serve/:file_id", r.APIService.ServeFile)
	r.GET("/stream/:file_id", r.APIService.StreamFile)
}

// RegisterAPIRoutes sets up API endpoints
func (r *Router) registerAPIRoutes() {
	{
		r.Use(r.APIGuardMiddleware())
		{
			r.POST("/upload", r.APIService.UploadFile)
			r.GET("/download/:file_id", r.APIService.DownloadFile)
			r.DELETE("/delete/:file_id", r.APIService.DeleteFile)
			r.DELETE("/scrub/:file_id", r.APIService.DeleteFilePermanently)
		}

		{
			r.POST("/new_folder", r.APIService.CreateFolder)
			r.GET("/folder_content/:folder_id", r.APIService.GetFolderContent)
			// r.GET("/folder_folders", r.APIService.GetSubFolders)
			// r.GET("/folder_files", r.APIService.GetFilesInFolder)
			// r.GET("/folder_descendants", r.APIService.GetDescendants)
			r.PUT("/rename_folder", r.APIService.RenameFolder)
			r.PUT("/move_folder", r.APIService.MoveFolder)
			r.DELETE("/delete_folder/:folder_id", r.APIService.DeleteFolder)
			r.DELETE("/scrub_folder/:folder_id", r.APIService.DeleteFolderPermanently)
		}
	}
}

// RegisterWebRoutes sets up the web interface routes
func (r *Router) registerWebRoutes() {
	/* Web Routes */
	web := r.Group("/web")
	{
		web.Use(r.WebGuardMiddleware())
		{
			web.GET("/", r.WebService.ViewFolder)
			web.GET("/folder/:folder_id", r.WebService.ViewFolder)
			web.POST("/new-folder", r.WebService.NewFolder)
			web.PUT("/rename-folder", r.WebService.RenameFolder)
			web.PUT("/move-folder", r.WebService.MoveFolder)
			web.DELETE("/folder/:folder_id", r.WebService.DeleteFolder)
			web.DELETE("/scrub-folder/:folder_id", r.WebService.DeleteFolderPermanently)

			web.POST("/upload", r.WebService.UploadFile)
			web.GET("/file/:file_id", r.WebService.DownloadFile)
			web.PUT("/file/:file_id", r.WebService.MoveFile)
			web.DELETE("/file/:file_id", r.WebService.DeleteFile)
			web.DELETE("/scrub/:file_id", r.WebService.DeleteFilePermanently)
		}
	}
}

// registerAllRoutes registers all required routes
func (r *Router) registerAllRoutes(StandaloneMode bool) {
	// Register core routes
	r.registerBaseRoutes()

	// Register web routes **only if in standalone mode**
	if StandaloneMode {
		r.registerWebRoutes()
	}

	// Register API routes
	r.registerAPIRoutes()
}
