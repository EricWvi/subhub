package provider

import (
	"context"
	"database/sql"
	"errors"
	"time"
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
