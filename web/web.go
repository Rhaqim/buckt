package web

import (
	"fmt"

	mainDomain "github.com/Rhaqim/buckt/internal/domain"
	mainWeb "github.com/Rhaqim/buckt/internal/web"
	"github.com/Rhaqim/buckt/pkg/logger"
	"github.com/Rhaqim/buckt/web/app"
	"github.com/Rhaqim/buckt/web/domain"
	"github.com/Rhaqim/buckt/web/middleware"
	"github.com/Rhaqim/buckt/web/router"
)

func NewRouterService(bucktLog *logger.BucktLogger, standaloneMode, debug bool, fileService mainDomain.FileService, folderService mainDomain.FolderService) (mainDomain.RouterService, error) {
	// Load templates
	bucktLog.Info("🚀 Loading templates")
	tmpl, err := loadTemplates()
	if err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	// Initialize the app services
	var apiService domain.APIService = app.NewAPIService(folderService, fileService)
	var webService domain.WebService = app.NewWebService(folderService, fileService)

	// middleware server
	var middleware domain.Middleware = middleware.NewBucketMiddleware(bucktLog, standaloneMode)

	// Run the router
	router := router.NewRouter(
		bucktLog, tmpl,
		debug,
		standaloneMode,
		apiService, webService, middleware)

	return router, nil
}

func init() {
	mainWeb.RegisterRouterInitialiser(NewRouterService)
}
