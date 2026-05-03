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
	assert.Equal(t, 4, appliedCount)

	rows, err := db.Query(`SELECT filename, applied_at FROM migrations ORDER BY filename`)
	require.NoError(t, err)
	defer rows.Close()
	var filenames []string
	for rows.Next() {
		var filename, appliedAt string
		require.NoError(t, rows.Scan(&filename, &appliedAt))
		filenames = append(filenames, filename)
		assert.NotEmpty(t, appliedAt)
	}
	require.NoError(t, rows.Err())
	assert.Equal(t, []string{"001_initial.sql", "002_add_rules.sql", "003_add_subscriptions.sql", "004_add_clash_config_proxy_group_position.sql"}, filenames)

	require.NoError(t, db.Close())

	reopened := store.MustOpen(dbPath)
	t.Cleanup(func() { reopened.Close() })

	err = reopened.QueryRow(`SELECT COUNT(*) FROM migrations`).Scan(&appliedCount)
	require.NoError(t, err)
	assert.Equal(t, 4, appliedCount)
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

func TestMustOpenBackfillsClashConfigProxyGroupPositionByPriorIDOrderPerSubscription(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")

	rawDB, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { rawDB.Close() })

	_, err = rawDB.Exec(`PRAGMA foreign_keys = ON`)
	require.NoError(t, err)

	_, err = rawDB.Exec(`
CREATE TABLE IF NOT EXISTS migrations (
	filename TEXT PRIMARY KEY,
	applied_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS providers (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	url TEXT NOT NULL,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS proxy_groups (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL UNIQUE,
	script TEXT NOT NULL,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS clash_config_subscriptions (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS clash_config_proxy_groups (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	subscription_id INTEGER NOT NULL,
	name TEXT NOT NULL,
	type TEXT NOT NULL,
	url TEXT NOT NULL DEFAULT '',
	interval_seconds INTEGER NOT NULL DEFAULT 0,
	bind_internal_proxy_group_id INTEGER,
	is_system INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	UNIQUE (subscription_id, name),
	UNIQUE (subscription_id, bind_internal_proxy_group_id),
	FOREIGN KEY(subscription_id) REFERENCES clash_config_subscriptions(id) ON DELETE CASCADE,
	FOREIGN KEY(bind_internal_proxy_group_id) REFERENCES proxy_groups(id)
);
`)
	require.NoError(t, err)

	_, err = rawDB.Exec(`
INSERT INTO clash_config_subscriptions (id, name, created_at, updated_at) VALUES
	(1, 'Daily', '2026-05-03T09:00:00+08:00', '2026-05-03T09:00:00+08:00'),
	(2, 'Nightly', '2026-05-03T09:00:00+08:00', '2026-05-03T09:00:00+08:00');

INSERT INTO clash_config_proxy_groups
	(id, subscription_id, name, type, url, interval_seconds, bind_internal_proxy_group_id, is_system, created_at, updated_at)
VALUES
	(10, 1, 'First', 'select', '', 0, NULL, 0, '2026-05-03T09:00:00+08:00', '2026-05-03T09:00:00+08:00'),
	(20, 1, 'Second', 'select', '', 0, NULL, 0, '2026-05-03T09:00:00+08:00', '2026-05-03T09:00:00+08:00'),
	(30, 1, 'Third', 'select', '', 0, NULL, 0, '2026-05-03T09:00:00+08:00', '2026-05-03T09:00:00+08:00'),
	(15, 2, 'Alpha', 'select', '', 0, NULL, 0, '2026-05-03T09:00:00+08:00', '2026-05-03T09:00:00+08:00'),
	(25, 2, 'Beta', 'select', '', 0, NULL, 0, '2026-05-03T09:00:00+08:00', '2026-05-03T09:00:00+08:00');

INSERT INTO migrations (filename, applied_at) VALUES
	('001_initial.sql', '2026-05-03T09:00:00+08:00'),
	('002_add_rules.sql', '2026-05-03T09:00:00+08:00'),
	('003_add_subscriptions.sql', '2026-05-03T09:00:00+08:00');
`)
	require.NoError(t, err)
	require.NoError(t, rawDB.Close())

	db := store.MustOpen(dbPath)
	t.Cleanup(func() { db.Close() })

	rows, err := db.Query(`
SELECT subscription_id, id, position
FROM clash_config_proxy_groups
ORDER BY subscription_id, position, id
`)
	require.NoError(t, err)
	defer rows.Close()

	type row struct {
		subscriptionID int64
		id             int64
		position       int64
	}

	var got []row
	for rows.Next() {
		var item row
		require.NoError(t, rows.Scan(&item.subscriptionID, &item.id, &item.position))
		got = append(got, item)
	}
	require.NoError(t, rows.Err())

	assert.Equal(t, []row{
		{subscriptionID: 1, id: 10, position: 0},
		{subscriptionID: 1, id: 20, position: 1},
		{subscriptionID: 1, id: 30, position: 2},
		{subscriptionID: 2, id: 15, position: 0},
		{subscriptionID: 2, id: 25, position: 1},
	}, got)
}
