package integration

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/EricWvi/subhub/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMustOpenRecordsAppliedMigrationsOnce(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")

	db := store.MustOpen(dbPath)
	t.Cleanup(func() { db.Close() })

	var appliedCount int
	err := db.QueryRow(`SELECT COUNT(*) FROM migrations`).Scan(&appliedCount)
	require.NoError(t, err)
	assert.Equal(t, 1, appliedCount)

	var filename string
	var appliedAt string
	err = db.QueryRow(`SELECT filename, applied_at FROM migrations`).Scan(&filename, &appliedAt)
	require.NoError(t, err)
	assert.Equal(t, "001_initial.sql", filename)
	assert.NotEmpty(t, appliedAt)

	require.NoError(t, db.Close())

	reopened := store.MustOpen(dbPath)
	t.Cleanup(func() { reopened.Close() })

	err = reopened.QueryRow(`SELECT COUNT(*) FROM migrations`).Scan(&appliedCount)
	require.NoError(t, err)
	assert.Equal(t, 1, appliedCount)
}

func TestMustOpenCreatesMigrationHistoryTable(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")

	db := store.MustOpen(dbPath)
	t.Cleanup(func() { db.Close() })

	var tableName string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'migrations'`).Scan(&tableName)
	require.NoError(t, err)
	assert.Equal(t, "migrations", tableName)
}

func TestMustOpenSkipsAlreadyAppliedMigrationWhenTableExists(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")

	db := store.MustOpen(dbPath)
	require.NoError(t, db.Close())

	rawDB, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { rawDB.Close() })

	_, err = rawDB.Exec(`DELETE FROM migrations`)
	require.NoError(t, err)
	_, err = rawDB.Exec(`INSERT INTO migrations (filename, applied_at) VALUES ('001_initial.sql', '2026-05-02T08:00:00+08:00')`)
	require.NoError(t, err)
	require.NoError(t, rawDB.Close())

	reopened := store.MustOpen(dbPath)
	t.Cleanup(func() { reopened.Close() })

	var appliedCount int
	err = reopened.QueryRow(`SELECT COUNT(*) FROM migrations WHERE filename = '001_initial.sql'`).Scan(&appliedCount)
	require.NoError(t, err)
	assert.Equal(t, 1, appliedCount)
}
