package model

type WebMode int

const (
	WebModeAll WebMode = iota
	WebModeAPI
	WebModeUI
	WebModeMount
)

func (wm WebMode) String() string {
	switch wm {
	case WebModeAPI:
		return "API"
	case WebModeUI:
		return "UI"
	case WebModeMount:
		return "Mount"
	default:
		return "All"
	}
}
