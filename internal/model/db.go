package model

type DBDrivers string

const (
	Postgres DBDrivers = "postgres"
	SQLite   DBDrivers = "sqlite"
)


// Pagination holds pagination parameters for list queries.
type Pagination struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// DefaultPagination returns sensible defaults.
func DefaultPagination() Pagination {
	return Pagination{Page: 1, PageSize: 50}
}

// Validate ensures pagination values are within acceptable bounds.
func (p *Pagination) Validate() {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize < 1 {
		p.PageSize = 50
	}
	if p.PageSize > 500 {
		p.PageSize = 500
	}
}

// Offset returns the SQL offset for the current page.
func (p Pagination) Offset() int {
	return (p.Page - 1) * p.PageSize
}
