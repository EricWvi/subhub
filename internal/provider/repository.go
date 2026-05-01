package provider

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"gopkg.in/yaml.v3"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, p Provider) (Provider, error) {
	now := time.Now().UTC()
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO providers (name, url, refresh_interval_seconds, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		p.Name, p.URL, p.RefreshIntervalSeconds, now.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	if err != nil {
		return Provider{}, err
	}
	id, _ := result.LastInsertId()
	p.ID = id
	p.CreatedAt = now
	p.UpdatedAt = now
	return p, nil
}

func (r *Repository) List(ctx context.Context) ([]Provider, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, url, refresh_interval_seconds, created_at, updated_at FROM providers ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var providers []Provider
	for rows.Next() {
		var p Provider
		var createdAt, updatedAt string
		if err := rows.Scan(&p.ID, &p.Name, &p.URL, &p.RefreshIntervalSeconds, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		p.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		p.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		providers = append(providers, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return providers, nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (Provider, error) {
	var p Provider
	var createdAt, updatedAt string
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, url, refresh_interval_seconds, created_at, updated_at FROM providers WHERE id = ?`, id,
	).Scan(&p.ID, &p.Name, &p.URL, &p.RefreshIntervalSeconds, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Provider{}, ErrNotFound
		}
		return Provider{}, err
	}
	p.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	p.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return p, nil
}

func (r *Repository) Update(ctx context.Context, p Provider) (Provider, error) {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx,
		`UPDATE providers SET name = ?, url = ?, refresh_interval_seconds = ?, updated_at = ? WHERE id = ?`,
		p.Name, p.URL, p.RefreshIntervalSeconds, now.Format(time.RFC3339), p.ID,
	)
	if err != nil {
		return Provider{}, err
	}
	p.UpdatedAt = now
	return p, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM providers WHERE id = ?`, id)
	return err
}

func (r *Repository) ReplaceLastKnownGoodSnapshot(ctx context.Context, providerID int64, format string, raw []byte, nodes []map[string]any) error {
	normalizedYAML, err := yaml.Marshal(map[string]any{"proxies": nodes})
	if err != nil {
		return err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `UPDATE provider_snapshots SET is_last_known_good = 0 WHERE provider_id = ?`, providerID); err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO provider_snapshots (provider_id, format, raw_payload, normalized_yaml, node_count, fetched_at, is_last_known_good) VALUES (?, ?, ?, ?, ?, ?, 1)`,
		providerID, format, raw, string(normalizedYAML), len(nodes), time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) RecordRefreshFailure(ctx context.Context, providerID int64, message string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO refresh_attempts (provider_id, status, message, attempted_at) VALUES (?, 'failure', ?, ?)`,
		providerID, message, time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

func (r *Repository) GetLatestSnapshot(ctx context.Context, providerID int64) (Snapshot, error) {
	var s Snapshot
	var fetchedAt string
	err := r.db.QueryRowContext(ctx,
		`SELECT id, provider_id, format, raw_payload, normalized_yaml, node_count, fetched_at, is_last_known_good FROM provider_snapshots WHERE provider_id = ? AND is_last_known_good = 1`,
		providerID,
	).Scan(&s.ID, &s.ProviderID, &s.Format, &s.RawPayload, &s.NormalizedYAML, &s.NodeCount, &fetchedAt, &s.IsLastKnownGood)
	if err != nil {
		return Snapshot{}, err
	}
	s.FetchedAt, _ = time.Parse(time.RFC3339, fetchedAt)
	return s, nil
}
