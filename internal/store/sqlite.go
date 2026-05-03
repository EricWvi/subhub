package store

import (
	"database/sql"
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

const migrationHistoryTableSQL = `
CREATE TABLE IF NOT EXISTS migrations (
	filename TEXT PRIMARY KEY,
	applied_at TEXT NOT NULL
);
`

func MustOpen(dbPath string) *sql.DB {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Fatalf("create db directory: %v", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		db.Close()
		log.Fatalf("enable foreign keys: %v", err)
	}

	if err := migrate(db); err != nil {
		db.Close()
		log.Fatalf("migrate database: %v", err)
	}

	return db
}

func migrate(db *sql.DB) error {
	if _, err := db.Exec(migrationHistoryTableSQL); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}

	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return fmt.Errorf("load timezone: %w", err)
	}

	for _, entry := range entries {
		applied, err := migrationApplied(db, entry.Name())
		if err != nil {
			return fmt.Errorf("check migration %s: %w", entry.Name(), err)
		}
		if applied {
			continue
		}

		data, err := migrationFS.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", entry.Name(), err)
		}

		skipExec, err := shouldSkipMigration(tx, entry.Name())
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("prepare migration %s: %w", entry.Name(), err)
		}
		if !skipExec {
			if _, err := tx.Exec(string(data)); err != nil {
				tx.Rollback()
				return fmt.Errorf("exec migration %s: %w", entry.Name(), err)
			}
		}

		appliedAt := time.Now().In(location).Format(time.RFC3339)
		if _, err := tx.Exec(`INSERT INTO migrations (filename, applied_at) VALUES (?, ?)`, entry.Name(), appliedAt); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %s: %w", entry.Name(), err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", entry.Name(), err)
		}
	}

	return nil
}

func migrationApplied(db *sql.DB, filename string) (bool, error) {
	var exists int
	err := db.QueryRow(`SELECT 1 FROM migrations WHERE filename = ?`, filename).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func shouldSkipMigration(tx *sql.Tx, filename string) (bool, error) {
	if filename != "004_add_clash_config_proxy_group_position.sql" {
		return false, nil
	}

	rows, err := tx.Query(`PRAGMA table_info(clash_config_proxy_groups)`)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var dataType string
		var notNull int
		var defaultValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			return false, err
		}
		if strings.EqualFold(name, "position") {
			return true, nil
		}
	}

	return false, rows.Err()
}
