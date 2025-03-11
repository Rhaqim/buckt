package database_test

import (
	"database/sql"
	"testing"

	"github.com/Rhaqim/buckt/internal/database"
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/pkg/logger"
	"github.com/stretchr/testify/assert"
)

func TestNewDB(t *testing.T) {
	log := logger.NewLogger("", true, false)

	t.Run("Unsupported driver falls back to SQLite", func(t *testing.T) {
		db, err := database.NewDB(nil, "unsupported", log, true)
		assert.NoError(t, err)
		assert.NotNil(t, db)
		assert.Equal(t, "sqlite", db.Dialector.Name())
	})

	t.Run("SQLite with provided instance", func(t *testing.T) {
		sqlDB, err := sql.Open("sqlite3", ":memory:")
		assert.NoError(t, err)
		defer sqlDB.Close()

		db, err := database.NewDB(sqlDB, domain.SQLite, log, true)
		assert.NoError(t, err)
		assert.NotNil(t, db)
		assert.Equal(t, "sqlite", db.Dialector.Name())
	})

	// t.Run("Postgres with provided instance", func(t *testing.T) {
	// 	sqlDB, err := sql.Open("postgres", "user=postgres password=postgres dbname=postgres sslmode=disable")
	// 	assert.NoError(t, err)
	// 	defer sqlDB.Close()

	// 	db, err := database.NewDB(sqlDB, domain.Postgres, log, true)
	// 	assert.NoError(t, err)
	// 	assert.NotNil(t, db)
	// 	assert.Equal(t, "postgres", db.Dialector.Name())
	// })

	t.Run("SQLite without provided instance", func(t *testing.T) {
		db, err := database.NewDB(nil, domain.SQLite, log, true)
		assert.NoError(t, err)
		assert.NotNil(t, db)
		assert.Equal(t, "sqlite", db.Dialector.Name())
	})
}

func TestDB_Close(t *testing.T) {
	log := logger.NewLogger("", true, false)
	sqlDB, err := sql.Open("sqlite3", ":memory:")
	assert.NoError(t, err)
	defer sqlDB.Close()

	db, err := database.NewDB(sqlDB, domain.SQLite, log, true)
	assert.NoError(t, err)
	assert.NotNil(t, db)

	err = db.Close()
	assert.NoError(t, err)
}

func TestDB_Migrate(t *testing.T) {
	log := logger.NewLogger("", true, false)
	sqlDB, err := sql.Open("sqlite3", ":memory:")
	assert.NoError(t, err)
	defer sqlDB.Close()

	db, err := database.NewDB(sqlDB, domain.SQLite, log, true)
	assert.NoError(t, err)
	assert.NotNil(t, db)

	err = db.Migrate()
	assert.NoError(t, err)
}
