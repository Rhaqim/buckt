package web

import (
	"fmt"
	"log"
	"os"

	"github.com/Rhaqim/buckt"
	"github.com/Rhaqim/buckt/client/web/app"
	"github.com/Rhaqim/buckt/client/web/domain"
	"github.com/Rhaqim/buckt/client/web/middleware"
	"github.com/Rhaqim/buckt/client/web/router"
)

func NewClient(bucktClient *buckt.Client, conf ...Config) (domain.RouterService, error) {
	logger := log.New(os.Stdout, "client: ", log.LstdFlags)

	tmpl, err := loadTemplates()
	if err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	apiService := app.NewAPIService(bucktClient)
	webService := app.NewWebService(bucktClient)

	mode := WebModeAll
	debug := false

	// Apply any provided configuration options
	for _, c := range conf {
		mode = c.Mode
		debug = c.Debug
	}

	// 	// middleware server
	mw := middleware.NewBucketMiddleware(logger, mode == WebModeMount)

	router := router.NewRouter(
		logger,
		tmpl,
		debug,
		mode,
		apiService,
		webService,
		mw)

	return router, nil
}
