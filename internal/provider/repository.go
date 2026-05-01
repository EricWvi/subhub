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

func nowInLocation() time.Time {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	return time.Now().In(loc)
}

func (r *Repository) Create(ctx context.Context, p Provider) (Provider, error) {
	now := nowInLocation()
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO providers (name, url, refresh_interval_minutes, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		p.Name, p.URL, p.RefreshIntervalMinutes, now.Format(time.RFC3339), now.Format(time.RFC3339),
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
		`SELECT 
			p.id, p.name, p.url, p.refresh_interval_minutes, p.created_at, p.updated_at,
			ra.status, ra.message
		FROM providers p
		LEFT JOIN refresh_attempts ra ON p.id = ra.provider_id
		ORDER BY p.id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var providers []Provider
	for rows.Next() {
		var p Provider
		var createdAt, updatedAt string
		var status, message sql.NullString
		if err := rows.Scan(&p.ID, &p.Name, &p.URL, &p.RefreshIntervalMinutes, &createdAt, &updatedAt, &status, &message); err != nil {
			return nil, err
		}
		p.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		p.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		p.LastRefreshStatus = status.String
		p.LastRefreshMessage = message.String
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
	var status, message sql.NullString
	err := r.db.QueryRowContext(ctx,
		`SELECT 
			p.id, p.name, p.url, p.refresh_interval_minutes, p.created_at, p.updated_at,
			ra.status, ra.message
		FROM providers p
		LEFT JOIN refresh_attempts ra ON p.id = ra.provider_id
		WHERE p.id = ?`, id,
	).Scan(&p.ID, &p.Name, &p.URL, &p.RefreshIntervalMinutes, &createdAt, &updatedAt, &status, &message)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Provider{}, ErrNotFound
		}
		return Provider{}, err
	}
	p.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	p.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	p.LastRefreshStatus = status.String
	p.LastRefreshMessage = message.String
	return p, nil
}

func (r *Repository) Update(ctx context.Context, p Provider) (Provider, error) {
	now := nowInLocation()
	_, err := r.db.ExecContext(ctx,
		`UPDATE providers SET name = ?, url = ?, refresh_interval_minutes = ?, updated_at = ? WHERE id = ?`,
		p.Name, p.URL, p.RefreshIntervalMinutes, now.Format(time.RFC3339), p.ID,
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

func (r *Repository) ReplaceLastKnownGoodSnapshot(ctx context.Context, providerID int64, format string, nodes []map[string]any) error {
	normalizedYAML, err := yaml.Marshal(map[string]any{"proxies": nodes})
	if err != nil {
		return err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := nowInLocation()
	nowStr := now.Format(time.RFC3339)

	_, err = tx.ExecContext(ctx,
		`INSERT INTO provider_snapshots (provider_id, format, normalized_yaml, node_count, fetched_at) 
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(provider_id) DO UPDATE SET 
		 	format=excluded.format, 
			normalized_yaml=excluded.normalized_yaml, 
			node_count=excluded.node_count, 
			fetched_at=excluded.fetched_at`,
		providerID, format, string(normalizedYAML), len(nodes), nowStr,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO refresh_attempts (provider_id, status, message, attempted_at) VALUES (?, 'success', 'OK', ?)
		 ON CONFLICT(provider_id) DO UPDATE SET status=excluded.status, message=excluded.message, attempted_at=excluded.attempted_at`,
		providerID, nowStr,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `UPDATE providers SET updated_at = ? WHERE id = ?`, nowStr, providerID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) RecordRefreshFailure(ctx context.Context, providerID int64, message string) error {
	now := nowInLocation()
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO refresh_attempts (provider_id, status, message, attempted_at) VALUES (?, 'failure', ?, ?)
		 ON CONFLICT(provider_id) DO UPDATE SET status=excluded.status, message=excluded.message, attempted_at=excluded.attempted_at`,
		providerID, message, now.Format(time.RFC3339),
	)
	return err
}

func (r *Repository) GetLatestSnapshot(ctx context.Context, providerID int64) (Snapshot, error) {
	var s Snapshot
	var fetchedAt string
	err := r.db.QueryRowContext(ctx,
		`SELECT id, provider_id, format, normalized_yaml, node_count, fetched_at FROM provider_snapshots WHERE provider_id = ?`,
		providerID,
	).Scan(&s.ID, &s.ProviderID, &s.Format, &s.NormalizedYAML, &s.NodeCount, &fetchedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Snapshot{}, ErrNotFound
		}
		return Snapshot{}, err
	}
	s.FetchedAt, _ = time.Parse(time.RFC3339, fetchedAt)
	return s, nil
}

func (r *Repository) ListLatestNodes(ctx context.Context) ([]map[string]any, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT normalized_yaml FROM provider_snapshots`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var all []map[string]any
	for rows.Next() {
		var normalized string
		if err := rows.Scan(&normalized); err != nil {
			return nil, err
		}
		var doc map[string]any
		if err := yaml.Unmarshal([]byte(normalized), &doc); err != nil {
			return nil, err
		}
		proxies, ok := doc["proxies"].([]any)
		if !ok {
			continue
		}
		for _, p := range proxies {
			if m, ok := p.(map[string]any); ok {
				all = append(all, m)
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return all, nil
}
