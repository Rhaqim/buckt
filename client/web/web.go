package web

import (
	"fmt"
	"log"
	"os"

	"github.com/Rhaqim/buckt"
	"github.com/Rhaqim/buckt/web/app"
	"github.com/Rhaqim/buckt/web/domain"
	"github.com/Rhaqim/buckt/web/middleware"
	"github.com/Rhaqim/buckt/web/router"
)

type ClientConfig struct {
	mode  WebMode
	debug bool
}

func NewClient(bucktClient *buckt.Client, conf ...ClientConfig) (domain.RouterService, error) {
	var logger *log.Logger = log.New(os.Stdout, "client: ", log.LstdFlags)

	tmpl, err := loadTemplates()
	if err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	var apiService domain.APIService = app.NewAPIService(bucktClient)
	var webService domain.WebService = app.NewWebService(bucktClient)

	mode := WebModeAll
	debug := false

	// Apply any provided configuration options
	for _, c := range conf {
		mode = c.mode
		debug = c.debug
	}

	// 	// middleware server
	var middleware domain.Middleware = middleware.NewBucketMiddleware(logger, mode == WebModeMount)

	router := router.NewRouter(
		logger,
		tmpl,
		debug,
		mode,
		apiService,
		webService,
		middleware)

	return router, nil
}
