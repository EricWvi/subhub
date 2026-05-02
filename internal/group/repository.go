package group

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrNotFound = errors.New("proxy group not found")

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

func (r *Repository) Create(ctx context.Context, g ProxyGroup) (ProxyGroup, error) {
	now := nowInLocation()
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO proxy_groups (name, script_text, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		g.Name, g.Script, now.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	if err != nil {
		return ProxyGroup{}, err
	}
	id, _ := result.LastInsertId()
	g.ID = id
	g.CreatedAt = now
	g.UpdatedAt = now
	return g, nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (ProxyGroup, error) {
	var g ProxyGroup
	var createdAt, updatedAt string
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, script_text, created_at, updated_at FROM proxy_groups WHERE id = ?`, id,
	).Scan(&g.ID, &g.Name, &g.Script, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ProxyGroup{}, ErrNotFound
		}
		return ProxyGroup{}, err
	}
	g.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	g.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return g, nil
}

func (r *Repository) List(ctx context.Context) ([]ProxyGroup, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, script_text, created_at, updated_at FROM proxy_groups ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []ProxyGroup
	for rows.Next() {
		var g ProxyGroup
		var createdAt, updatedAt string
		if err := rows.Scan(&g.ID, &g.Name, &g.Script, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		g.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		g.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

func (r *Repository) Update(ctx context.Context, g ProxyGroup) (ProxyGroup, error) {
	now := nowInLocation()
	_, err := r.db.ExecContext(ctx,
		`UPDATE proxy_groups SET name = ?, script_text = ?, updated_at = ? WHERE id = ?`,
		g.Name, g.Script, now.Format(time.RFC3339), g.ID,
	)
	if err != nil {
		return ProxyGroup{}, err
	}
	g.UpdatedAt = now
	return g, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM proxy_groups WHERE id = ?`, id)
	return err
}

func (r *Repository) ListProxyNodeViews(ctx context.Context) ([]ProxyNodeView, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT n.id, p.name, n.name
		 FROM proxy_nodes n
		 JOIN providers p ON p.id = n.provider_id
		 ORDER BY p.id, n.id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []ProxyNodeView
	for rows.Next() {
		var node ProxyNodeView
		if err := rows.Scan(&node.ID, &node.ProviderName, &node.Name); err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	return nodes, rows.Err()
}

func (r *Repository) ListRawNodesByProviders(ctx context.Context, providerIDs []int64) ([]ResolvedNode, error) {
	if len(providerIDs) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(providerIDs))
	args := make([]any, len(providerIDs))
	for i, id := range providerIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	query := fmt.Sprintf(
		`SELECT n.id, n.provider_id, n.name, n.raw_yaml FROM proxy_nodes n WHERE n.provider_id IN (%s) ORDER BY n.provider_id, n.id`,
		strings.Join(placeholders, ","),
	)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []ResolvedNode
	for rows.Next() {
		var n ResolvedNode
		if err := rows.Scan(&n.ID, &n.ProviderID, &n.Name, &n.RawYAML); err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}
