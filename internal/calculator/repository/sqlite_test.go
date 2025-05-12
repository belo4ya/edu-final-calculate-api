package repository

import (
	"path/filepath"
	"testing"

	"github.com/belo4ya/edu-final-calculate-api/internal/calculator/database"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t testing.TB) *sqlx.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := database.Connect(t.Context(), dbPath)
	require.NoError(t, err)

	driver, err := sqlite3.WithInstance(db.DB, &sqlite3.Config{})
	require.NoError(t, err)

	m, err := migrate.NewWithDatabaseInstance("file://../../../migrations", "sqlite3", driver)
	require.NoError(t, err)

	err = m.Up()
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}
