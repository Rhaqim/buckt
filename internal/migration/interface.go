package migration

import (
	"time"
)

type MigrationMode int

const (
	MigrateModeNone          MigrationMode = iota
	MigrateModeToSecondary                 // primary --> secondary (e.g., local -> s3)
	MigrateModeFromSecondary               // secondary --> primary (e.g., s3 -> local)
)

type MigrationConfig struct {
	Concurrency     int
	RetryCount      int
	RetryBackoff    time.Duration
	DeleteAfterCopy bool   // remove source after successful migration
	PersistPath     string // where to store checkpoint file if primary is local
}

type migrationState struct {
	Prefix    string          `json:"prefix"`
	Processed map[string]bool `json:"processed"`
	Total     int64           `json:"total"`
	Completed int64           `json:"completed"`
	StartedAt time.Time       `json:"started_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}
