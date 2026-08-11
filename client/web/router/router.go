package router

import (
	"html/template"
	"log"
	"net/http"

	"github.com/Rhaqim/buckt/client/web/domain"
	"github.com/Rhaqim/buckt/client/web/model"
	"github.com/gin-gonic/gin"
)

type Router struct {
	*gin.Engine

	mode model.WebMode

	domain.APIService
	domain.WebService
	domain.Middleware
}

// NewRouter creates a new router with the given logger and config.
func NewRouter(
	log *log.Logger,
	tmpl *template.Template,

	Debug bool,
	mode model.WebMode,

	apiService domain.APIService,
	webService domain.WebService,
	middleware domain.Middleware,
) domain.RouterService {
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

		mode: mode,

		APIService: apiService,
		WebService: webService,
		Middleware: middleware,
	}

	return router
}

// Run starts the router on the given address.
func (r *Router) Run(addr string) error {
	r.registerAllRoutes(r.mode)

	return r.Engine.Run(addr)
}

// ServeHTTP makes Router compatible with http.Handler
func (r *Router) Handler() http.Handler {
	r.registerAllRoutes(model.WebModeMount)

	return r.Engine
}

// RegisterAllRoutes registers all routes for the router.
func (r *Router) registerBaseRoutes(mode model.WebMode) {
	r.GET("/", func(c *gin.Context) {
		// redirect to /web
		c.Redirect(http.StatusMovedPermanently, "/web")
	})

	// /serve and /stream expose raw file *contents* by id. Browsers load these
	// directly (<img src>, <video>, download links) and cannot attach the
	// buckt-User-ID header, so guarding them with header auth breaks the bundled
	// UI — the symptom is a 401 when viewing an uploaded file. Scope the guard to
	// the mode instead:
	//   - UI / All: the bundled UI is single-tenant, so serve content as the
	//     default owner (consistent with the open /web routes).
	//   - API: standalone API callers can send the header, so require it.
	//   - Mount: the guard sets owner_id="default" and passes through; the
	//     mounting app applies its own auth upstream.
	// For per-object authorization in production, front buckt with signed URLs —
	// the bundled handlers still call the un-scoped core GetFile.
	serve := r.Group("/")
	switch mode {
	case model.WebModeAPI, model.WebModeMount:
		serve.Use(r.APIGuardMiddleware())
	default: // WebModeUI, WebModeAll — browser-facing, single-tenant
		serve.Use(r.WebGuardMiddleware())
	}
	{
		serve.GET("serve/:file_id", r.APIService.ServeFile)
		serve.GET("serve/:file_id/derivative/:name", r.APIService.ServeDerivative)
		serve.GET("stream/:file_id", r.APIService.StreamFile)
	}

	// Metrics dashboard endpoint (JSON): storage bytes, cache hit/miss, and
	// backend operation counts with R2 billing class. Surfaces the metrics added
	// in buckt 1.6.0.
	r.GET("/metrics", r.APIService.Metrics)

	// Which storage backend is active + migration progress (for the UI badge).
	r.GET("/backend", r.APIService.Backend)
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
			web.GET("/trash", r.WebService.ViewTrash)
			web.POST("/restore-file/:file_id", r.WebService.RestoreFile)
			web.POST("/restore-folder/:folder_id", r.WebService.RestoreFolder)
			web.POST("/new-folder", r.WebService.NewFolder)
			web.POST("/rename-folder", r.WebService.RenameFolder)
			web.POST("/move-folder", r.WebService.MoveFolder)
			web.DELETE("/folder/:folder_id", r.WebService.DeleteFolder)
			web.DELETE("/scrub-folder/:folder_id", r.WebService.DeleteFolderPermanently)

			web.POST("/upload", r.WebService.UploadFile)
			web.POST("/regenerate-derivatives/:file_id", r.WebService.RegenerateDerivatives)
			web.GET("/file/:file_id", r.WebService.DownloadFile)
			web.POST("/move-file/:file_id", r.WebService.MoveFile)
			web.DELETE("/file/:file_id", r.WebService.DeleteFile)
			web.DELETE("/scrub/:file_id", r.WebService.DeleteFilePermanently)
		}
	}
}

// registerAllRoutes registers all required routes
func (r *Router) registerAllRoutes(mode model.WebMode) {
	// Register core routes
	r.registerBaseRoutes(mode)

	switch mode {
	case model.WebModeAPI, model.WebModeMount:
		r.registerAPIRoutes()
	case model.WebModeUI:
		r.registerWebRoutes()
	default:
		r.registerWebRoutes()
		r.registerAPIRoutes()
	}
}
