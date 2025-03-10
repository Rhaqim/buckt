package buckt

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"strings"
)

//go:embed internal/web/templates/*.html
var templatesFS embed.FS

// loadTemplates loads and parses HTML templates from the embedded filesystem.
// It returns a parsed template or an error if the templates could not be loaded or parsed.
//
// The function first attempts to create a sub-filesystem from the embedded filesystem
// rooted at "internal/web/templates". If this operation fails, it returns an error
// indicating the failure to load templates.
//
// If the sub-filesystem is successfully created, the function then attempts to parse
// all HTML files within this sub-filesystem. If parsing fails, it returns an error
// indicating the failure to parse templates.
//
// Returns:
// - *template.Template: The parsed templates.
// - error: An error if the templates could not be loaded or parsed.
func loadTemplates() (*template.Template, error) {
	tmplFS, err := fs.Sub(templatesFS, "internal/web/templates")
	if err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	// Add custom functions to the template
	tmpl := template.New("").Funcs(template.FuncMap{
		"hasPrefix": hasPrefix,
	})

	tmpl, err = tmpl.ParseFS(tmplFS, "*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	return tmpl, nil
}

// Function to check if a string has a prefix
func hasPrefix(s, prefix string) bool {
	return strings.HasPrefix(s, prefix)
}
