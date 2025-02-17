package buckt

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
)

//go:embed internal/web/templates/*.html
var templatesFS embed.FS

func loadTemplates() (*template.Template, error) {
	tmplFS, err := fs.Sub(templatesFS, "internal/web/templates")
	if err != nil {
		// log.Fatal("Failed to load templates:", err)
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	tmpl, err := template.ParseFS(tmplFS, "*.html")
	if err != nil {
		// log.Fatal("Failed to parse templates:", err)
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	return tmpl, nil
}
