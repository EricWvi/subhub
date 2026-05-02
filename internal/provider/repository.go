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
		`INSERT INTO providers (name, url, refresh_interval_minutes, abbrev, used, total, expire, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.Name, p.URL, p.RefreshIntervalMinutes, p.Abbrev, p.Used, p.Total, p.Expire, now.Format(time.RFC3339), now.Format(time.RFC3339),
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
			p.id, p.name, p.url, p.refresh_interval_minutes, p.abbrev, p.used, p.total, p.expire, p.created_at, p.updated_at,
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
		if err := rows.Scan(&p.ID, &p.Name, &p.URL, &p.RefreshIntervalMinutes, &p.Abbrev, &p.Used, &p.Total, &p.Expire, &createdAt, &updatedAt, &status, &message); err != nil {
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
			p.id, p.name, p.url, p.refresh_interval_minutes, p.abbrev, p.used, p.total, p.expire, p.created_at, p.updated_at,
			ra.status, ra.message
		FROM providers p
		LEFT JOIN refresh_attempts ra ON p.id = ra.provider_id
		WHERE p.id = ?`, id,
	).Scan(&p.ID, &p.Name, &p.URL, &p.RefreshIntervalMinutes, &p.Abbrev, &p.Used, &p.Total, &p.Expire, &createdAt, &updatedAt, &status, &message)
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
		`UPDATE providers SET name = ?, url = ?, refresh_interval_minutes = ?, abbrev = ?, used = ?, total = ?, expire = ?, updated_at = ? WHERE id = ?`,
		p.Name, p.URL, p.RefreshIntervalMinutes, p.Abbrev, p.Used, p.Total, p.Expire, now.Format(time.RFC3339), p.ID,
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

type ReplaceSnapshotInput struct {
	Format       string
	Nodes        []map[string]any
	Used         int64
	Total        int64
	Expire       int64
	HasUsageInfo bool
}

func (r *Repository) ReplaceLastKnownGoodSnapshot(ctx context.Context, providerID int64, in ReplaceSnapshotInput) error {
	normalizedYAML, err := yaml.Marshal(map[string]any{"proxies": in.Nodes})
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
		providerID, in.Format, string(normalizedYAML), len(in.Nodes), nowStr,
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

	if in.HasUsageInfo {
		_, err = tx.ExecContext(ctx,
			`UPDATE providers SET used = ?, total = ?, expire = ?, updated_at = ? WHERE id = ?`,
			in.Used, in.Total, in.Expire, nowStr, providerID,
		)
	} else {
		_, err = tx.ExecContext(ctx,
			`UPDATE providers SET used = 0, total = 0, expire = 0, updated_at = ? WHERE id = ?`,
			nowStr, providerID,
		)
	}
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `UPDATE proxy_nodes SET update_mark = 1 WHERE provider_id = ?`, providerID)
	if err != nil {
		return err
	}

	for _, node := range in.Nodes {
		name, _ := node["name"].(string)
		raw, err := yaml.Marshal(node)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx,
			`INSERT INTO proxy_nodes (provider_id, name, raw_yaml, update_mark)
			 VALUES (?, ?, ?, 0)
			 ON CONFLICT(provider_id, name) DO UPDATE SET
			 	raw_yaml = excluded.raw_yaml,
			 	update_mark = 0`,
			providerID, name, string(raw),
		)
		if err != nil {
			return err
		}
	}

	_, err = tx.ExecContext(ctx, `DELETE FROM proxy_nodes WHERE provider_id = ? AND update_mark = 1`, providerID)
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

func (r *Repository) ListProxyNodesByProvider(ctx context.Context, providerID int64) ([]ProxyNode, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, provider_id, name, raw_yaml, update_mark
		 FROM proxy_nodes
		 WHERE provider_id = ?
		 ORDER BY id`,
		providerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []ProxyNode
	for rows.Next() {
		var n ProxyNode
		if err := rows.Scan(&n.ID, &n.ProviderID, &n.Name, &n.RawYAML, &n.UpdateMark); err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
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
		`SELECT raw_yaml FROM proxy_nodes ORDER BY provider_id, id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var all []map[string]any
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var node map[string]any
		if err := yaml.Unmarshal([]byte(raw), &node); err != nil {
			return nil, err
		}
		all = append(all, node)
	}
	return all, rows.Err()
}
