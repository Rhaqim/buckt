package web

import "github.com/Rhaqim/buckt/web/model"

type WebMode = model.WebMode

const (
	// WebModeAll registers all routes.
	WebModeAll = model.WebModeAll

	// WebModeAPI registers only the API routes.
	WebModeAPI = model.WebModeAPI

	// WebModeUI registers only the UI routes.
	WebModeUI = model.WebModeUI

	// WebModeMount registers only the API routes for the mount point.
	WebModeMount = model.WebModeMount
)

type ClientConfig struct {
	Mode  WebMode
	Debug bool
}
