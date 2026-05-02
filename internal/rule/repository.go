package rule

import (
	"context"
	"database/sql"
	"time"
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

func (r *Repository) List(ctx context.Context, page, pageSize int) ([]Rule, int, error) {
	var total int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM rules`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, rule_type, pattern, target_kind, proxy_group_id, created_at, updated_at FROM rules ORDER BY id LIMIT ? OFFSET ?`,
		pageSize, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var rules []Rule
	for rows.Next() {
		var rl Rule
		var targetKind string
		var proxyGroupID sql.NullInt64
		var createdAt, updatedAt string
		if err := rows.Scan(&rl.ID, &rl.RuleType, &rl.Pattern, &targetKind, &proxyGroupID, &createdAt, &updatedAt); err != nil {
			return nil, 0, err
		}
		rl.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		rl.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		rules = append(rules, rl)
	}
	return rules, total, rows.Err()
}
