package node

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var ErrNotFound = errors.New("node not found")

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

func (r *Repository) Create(ctx context.Context, n Node) (Node, error) {
	now := nowInLocation()
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO custom_nodes (name, config, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		n.Name, n.Config, now.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	if err != nil {
		return Node{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Node{}, err
	}
	n.ID = id
	n.CreatedAt = now
	n.UpdatedAt = now
	return n, nil
}

func (r *Repository) List(ctx context.Context) ([]Node, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, config, created_at, updated_at FROM custom_nodes ORDER BY id DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []Node
	for rows.Next() {
		var n Node
		var createdAt, updatedAt string
		if err := rows.Scan(&n.ID, &n.Name, &n.Config, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		n.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		n.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id int64) (Node, error) {
	var n Node
	var createdAt, updatedAt string
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, config, created_at, updated_at FROM custom_nodes WHERE id = ?`, id,
	).Scan(&n.ID, &n.Name, &n.Config, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Node{}, ErrNotFound
		}
		return Node{}, err
	}
	n.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	n.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return n, nil
}

func (r *Repository) Update(ctx context.Context, n Node) (Node, error) {
	now := nowInLocation()
	result, err := r.db.ExecContext(ctx,
		`UPDATE custom_nodes SET name = ?, config = ?, updated_at = ? WHERE id = ?`,
		n.Name, n.Config, now.Format(time.RFC3339), n.ID,
	)
	if err != nil {
		return Node{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return Node{}, err
	}
	if affected == 0 {
		return Node{}, ErrNotFound
	}
	n.UpdatedAt = now
	return n, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM custom_nodes WHERE id = ?`, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}
