package rule

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var ErrNotFound = errors.New("rule not found")

type ListRulesInput struct {
	Page     int
	PageSize int
	Search   string
}

func normalizeListInput(in ListRulesInput) ListRulesInput {
	if in.Page < 1 {
		in.Page = 1
	}
	if in.PageSize < 1 {
		in.PageSize = 20
	}
	if in.PageSize > 100 {
		in.PageSize = 100
	}
	return in
}

type CreateRuleRecord struct {
	RuleType     string
	Pattern      string
	TargetKind   string
	ProxyGroupID sql.NullInt64
}

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

func (r *Repository) FindProxyGroupIDByName(ctx context.Context, name string) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `SELECT id FROM proxy_groups WHERE name = ?`, name).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) BatchImport(ctx context.Context, records []CreateRuleRecord) (int, error) {
	now := nowInLocation()
	nowStr := now.Format(time.RFC3339)
	imported := 0
	for _, rec := range records {
		_, err := r.db.ExecContext(ctx,
			`INSERT INTO rules (rule_type, pattern, target_kind, proxy_group_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
			rec.RuleType, rec.Pattern, rec.TargetKind, rec.ProxyGroupID, nowStr, nowStr,
		)
		if err != nil {
			return imported, err
		}
		imported++
	}
	return imported, nil
}

func (r *Repository) Create(ctx context.Context, rec CreateRuleRecord) (Rule, error) {
	now := nowInLocation()
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO rules (rule_type, pattern, target_kind, proxy_group_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		rec.RuleType, rec.Pattern, rec.TargetKind, rec.ProxyGroupID, now.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	if err != nil {
		return Rule{}, err
	}
	id, _ := result.LastInsertId()
	return r.GetByID(ctx, id)
}

func (r *Repository) GetByID(ctx context.Context, id int64) (Rule, error) {
	var rl Rule
	var targetKind string
	var proxyGroupID sql.NullInt64
	var groupName sql.NullString
	var createdAt, updatedAt string

	err := r.db.QueryRowContext(ctx,
		`SELECT r.id, r.rule_type, r.pattern, r.target_kind, r.proxy_group_id, pg.name, r.created_at, r.updated_at
		 FROM rules r
		 LEFT JOIN proxy_groups pg ON pg.id = r.proxy_group_id
		 WHERE r.id = ?`,
		id,
	).Scan(&rl.ID, &rl.RuleType, &rl.Pattern, &targetKind, &proxyGroupID, &groupName, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Rule{}, ErrNotFound
		}
		return Rule{}, err
	}
	if targetKind == "PROXY_GROUP" {
		rl.ProxyGroup = groupName.String
	} else {
		rl.ProxyGroup = targetKind
	}
	rl.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	rl.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return rl, nil
}

func (r *Repository) List(ctx context.Context, in ListRulesInput) ([]Rule, int, error) {
	in = normalizeListInput(in)

	var countArgs []any
	countQuery := `SELECT COUNT(*) FROM rules`
	if in.Search != "" {
		countQuery += ` WHERE pattern LIKE ?`
		countArgs = append(countArgs, "%"+in.Search+"%")
	}

	var total int
	err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (in.Page - 1) * in.PageSize
	listQuery := `SELECT r.id, r.rule_type, r.pattern, r.target_kind, r.proxy_group_id, pg.name, r.created_at, r.updated_at
		 FROM rules r
		 LEFT JOIN proxy_groups pg ON pg.id = r.proxy_group_id`
	var listArgs []any
	if in.Search != "" {
		listQuery += ` WHERE r.pattern LIKE ?`
		listArgs = append(listArgs, "%"+in.Search+"%")
	}
	listQuery += ` ORDER BY r.id DESC LIMIT ? OFFSET ?`
	listArgs = append(listArgs, in.PageSize, offset)

	rows, err := r.db.QueryContext(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var rules []Rule
	for rows.Next() {
		var rl Rule
		var targetKind string
		var proxyGroupID sql.NullInt64
		var groupName sql.NullString
		var createdAt, updatedAt string
		if err := rows.Scan(&rl.ID, &rl.RuleType, &rl.Pattern, &targetKind, &proxyGroupID, &groupName, &createdAt, &updatedAt); err != nil {
			return nil, 0, err
		}
		if targetKind == "PROXY_GROUP" {
			rl.ProxyGroup = groupName.String
		} else {
			rl.ProxyGroup = targetKind
		}
		rl.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		rl.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		rules = append(rules, rl)
	}
	return rules, total, rows.Err()
}

func (r *Repository) Update(ctx context.Context, id int64, rec CreateRuleRecord) (Rule, error) {
	now := nowInLocation()
	_, err := r.db.ExecContext(ctx,
		`UPDATE rules SET rule_type = ?, pattern = ?, target_kind = ?, proxy_group_id = ?, updated_at = ? WHERE id = ?`,
		rec.RuleType, rec.Pattern, rec.TargetKind, rec.ProxyGroupID, now.Format(time.RFC3339), id,
	)
	if err != nil {
		return Rule{}, err
	}
	return r.GetByID(ctx, id)
}

func (r *Repository) ListForOutput(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT r.rule_type, r.pattern,
		        CASE WHEN r.target_kind = 'PROXY_GROUP' THEN pg.name ELSE r.target_kind END AS proxy_group
		 FROM rules r
		 LEFT JOIN proxy_groups pg ON pg.id = r.proxy_group_id
		 ORDER BY r.id DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []string
	for rows.Next() {
		var ruleType, pattern, proxyGroup string
		if err := rows.Scan(&ruleType, &pattern, &proxyGroup); err != nil {
			return nil, err
		}
		rules = append(rules, ruleType+","+pattern+","+proxyGroup)
	}
	return rules, rows.Err()
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM rules WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

type RuleOutputRow struct {
	ID             int64
	RuleType       string
	Pattern        string
	TargetKind     string
	ProxyGroupID   sql.NullInt64
	ProxyGroupName sql.NullString
}

func (r *Repository) ListForInternalGroup(ctx context.Context, groupID int64) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT r.rule_type, r.pattern,
		        CASE WHEN r.target_kind = 'PROXY_GROUP' THEN pg.name ELSE r.target_kind END AS proxy_group
		 FROM rules r
		 LEFT JOIN proxy_groups pg ON pg.id = r.proxy_group_id
		 WHERE r.proxy_group_id = ?
		 ORDER BY r.id DESC`, groupID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []string
	for rows.Next() {
		var ruleType, pattern, proxyGroup string
		if err := rows.Scan(&ruleType, &pattern, &proxyGroup); err != nil {
			return nil, err
		}
		rules = append(rules, ruleType+","+pattern+","+proxyGroup)
	}
	return rules, rows.Err()
}

func (r *Repository) ListAscendingForOutput(ctx context.Context) ([]RuleOutputRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT r.id, r.rule_type, r.pattern, r.target_kind, r.proxy_group_id, pg.name
		 FROM rules r
		 LEFT JOIN proxy_groups pg ON pg.id = r.proxy_group_id
		 ORDER BY r.id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []RuleOutputRow
	for rows.Next() {
		var r RuleOutputRow
		if err := rows.Scan(&r.ID, &r.RuleType, &r.Pattern, &r.TargetKind, &r.ProxyGroupID, &r.ProxyGroupName); err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}
	return rules, rows.Err()
}
