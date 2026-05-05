package subscription

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

func nowInLocation() time.Time {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	return time.Now().In(loc)
}

func (r *Repository) ListClashConfigs(ctx context.Context) ([]ClashConfigSubscription, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, created_at, updated_at FROM clash_config_subscriptions ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}

	var subs []ClashConfigSubscription
	for rows.Next() {
		var sub ClashConfigSubscription
		var createdAt, updatedAt string
		if err := rows.Scan(&sub.ID, &sub.Name, &createdAt, &updatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		sub.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		sub.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		subs = append(subs, sub)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	for i := range subs {
		providers, err := r.loadClashConfigProviders(ctx, subs[i].ID)
		if err != nil {
			return nil, err
		}
		subs[i].Providers = providers

		groups, err := r.loadClashConfigProxyGroups(ctx, subs[i].ID)
		if err != nil {
			return nil, err
		}
		subs[i].ProxyGroups = groups
	}
	return subs, nil
}

func (r *Repository) ListProxyProviders(ctx context.Context) ([]ProxyProviderSubscription, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, internal_proxy_group_id, created_at, updated_at FROM proxy_provider_subscriptions ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}

	var subs []ProxyProviderSubscription
	for rows.Next() {
		var sub ProxyProviderSubscription
		var createdAt, updatedAt string
		if err := rows.Scan(&sub.ID, &sub.Name, &sub.InternalProxyGroupID, &createdAt, &updatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		sub.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		sub.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		subs = append(subs, sub)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	for i := range subs {
		providers, err := r.loadSubscriptionProviders(ctx, "proxy_provider_subscription_providers", subs[i].ID)
		if err != nil {
			return nil, err
		}
		subs[i].Providers = providers
	}
	return subs, nil
}

func (r *Repository) ListRuleProviders(ctx context.Context) ([]RuleProviderSubscription, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, internal_proxy_group_id, created_at, updated_at FROM rule_provider_subscriptions ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}

	var subs []RuleProviderSubscription
	for rows.Next() {
		var sub RuleProviderSubscription
		var createdAt, updatedAt string
		if err := rows.Scan(&sub.ID, &sub.Name, &sub.InternalProxyGroupID, &createdAt, &updatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		sub.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		sub.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		subs = append(subs, sub)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	for i := range subs {
		providers, err := r.loadSubscriptionProviders(ctx, "rule_provider_subscription_providers", subs[i].ID)
		if err != nil {
			return nil, err
		}
		subs[i].Providers = providers
	}
	return subs, nil
}

func (r *Repository) CreateClashConfig(ctx context.Context, in CreateClashConfigSubscriptionInput) (ClashConfigSubscription, error) {
	var subID int64

	err := func() error {
		tx, err := r.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()

		now := nowInLocation()
		nowStr := now.Format(time.RFC3339)

		result, err := tx.ExecContext(ctx,
			`INSERT INTO clash_config_subscriptions (name, created_at, updated_at) VALUES (?, ?, ?)`,
			in.Name, nowStr, nowStr,
		)
		if err != nil {
			return err
		}
		subID, _ = result.LastInsertId()

		for i, pid := range in.Providers {
			_, err = tx.ExecContext(ctx,
				`INSERT INTO clash_config_subscription_providers (subscription_id, provider_id, position) VALUES (?, ?, ?)`,
				subID, pid, i,
			)
			if err != nil {
				return err
			}
		}

		for i, pg := range in.ProxyGroups {
			pg.Position = int64(i)
			_, err = insertProxyGroup(ctx, tx, subID, pg, nowStr)
			if err != nil {
				return err
			}
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		return nil
	}()
	if err != nil {
		return ClashConfigSubscription{}, err
	}

	return r.GetClashConfigByID(ctx, subID)
}

func insertProxyGroup(ctx context.Context, tx *sql.Tx, subID int64, pg CreateClashConfigProxyGroupInput, nowStr string) (int64, error) {
	isSystem := 0
	result, err := tx.ExecContext(ctx,
		`INSERT INTO clash_config_proxy_groups (subscription_id, name, type, position, url, interval_seconds, bind_internal_proxy_group_id, is_system, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		subID, pg.Name, pg.Type, pg.Position, pg.URL, pg.Interval, pg.BindInternalGroupID, isSystem, nowStr, nowStr,
	)
	if err != nil {
		return 0, err
	}
	pgID, _ := result.LastInsertId()

	for i, m := range pg.Proxies {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO clash_config_proxy_group_members (proxy_group_id, position, member_type, member_value) VALUES (?, ?, ?, ?)`,
			pgID, i, m.Type, m.Value,
		)
		if err != nil {
			return 0, err
		}
	}
	return pgID, nil
}

func (r *Repository) CreateSystemProxyGroup(ctx context.Context, subscriptionID int64, in CreateClashConfigProxyGroupInput) (ClashConfigProxyGroup, error) {
	var pgID int64

	err := func() error {
		tx, err := r.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()

		now := nowInLocation()
		nowStr := now.Format(time.RFC3339)

		result, err := tx.ExecContext(ctx,
			`INSERT INTO clash_config_proxy_groups (subscription_id, name, type, url, interval_seconds, bind_internal_proxy_group_id, is_system, created_at, updated_at) VALUES (?, ?, ?, ?, ?, NULL, 1, ?, ?)`,
			subscriptionID, in.Name, in.Type, in.URL, in.Interval, nowStr, nowStr,
		)
		if err != nil {
			return err
		}
		pgID, _ = result.LastInsertId()

		for i, m := range in.Proxies {
			_, err = tx.ExecContext(ctx,
				`INSERT INTO clash_config_proxy_group_members (proxy_group_id, position, member_type, member_value) VALUES (?, ?, ?, ?)`,
				pgID, i, m.Type, m.Value,
			)
			if err != nil {
				return err
			}
		}

		return tx.Commit()
	}()
	if err != nil {
		return ClashConfigProxyGroup{}, err
	}

	return ClashConfigProxyGroup{
		ID:                  pgID,
		Name:                in.Name,
		Type:                in.Type,
		Position:            0,
		URL:                 in.URL,
		Interval:            in.Interval,
		Proxies:             in.Proxies,
		BindInternalGroupID: in.BindInternalGroupID,
		IsSystem:            true,
	}, nil
}

func (r *Repository) GetClashConfigByID(ctx context.Context, id int64) (ClashConfigSubscription, error) {
	var sub ClashConfigSubscription

	err := func() error {
		tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
		if err != nil {
			return err
		}
		defer tx.Rollback()

		var createdAt, updatedAt string
		err = tx.QueryRowContext(ctx,
			`SELECT id, name, created_at, updated_at FROM clash_config_subscriptions WHERE id = ?`, id,
		).Scan(&sub.ID, &sub.Name, &createdAt, &updatedAt)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		sub.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		sub.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

		providersRows, err := tx.QueryContext(ctx,
			`SELECT provider_id FROM clash_config_subscription_providers WHERE subscription_id = ? ORDER BY position`, id,
		)
		if err != nil {
			return err
		}
		for providersRows.Next() {
			var pid int64
			if err := providersRows.Scan(&pid); err != nil {
				providersRows.Close()
				return err
			}
			sub.Providers = append(sub.Providers, pid)
		}
		providersRows.Close()
		if sub.Providers == nil {
			sub.Providers = []int64{}
		}

		pgRows, err := tx.QueryContext(ctx,
			`SELECT id, name, type, position, url, interval_seconds, bind_internal_proxy_group_id, is_system FROM clash_config_proxy_groups WHERE subscription_id = ? ORDER BY position, id`, id,
		)
		if err != nil {
			return err
		}

		var groups []ClashConfigProxyGroup
		for pgRows.Next() {
			var g ClashConfigProxyGroup
			var isSystem int64
			var bindID sql.NullInt64
			if err := pgRows.Scan(&g.ID, &g.Name, &g.Type, &g.Position, &g.URL, &g.Interval, &bindID, &isSystem); err != nil {
				pgRows.Close()
				return err
			}
			g.BindInternalGroupID = bindID.Int64
			g.IsSystem = isSystem == 1
			groups = append(groups, g)
		}
		pgRows.Close()

		for i := range groups {
			mRows, err := tx.QueryContext(ctx,
				`SELECT member_type, member_value FROM clash_config_proxy_group_members WHERE proxy_group_id = ? ORDER BY position`, groups[i].ID,
			)
			if err != nil {
				return err
			}
			for mRows.Next() {
				var m ProxyMember
				if err := mRows.Scan(&m.Type, &m.Value); err != nil {
					mRows.Close()
					return err
				}
				groups[i].Proxies = append(groups[i].Proxies, m)
			}
			mRows.Close()
			if groups[i].Proxies == nil {
				groups[i].Proxies = []ProxyMember{}
			}
		}
		sub.ProxyGroups = groups
		if sub.ProxyGroups == nil {
			sub.ProxyGroups = []ClashConfigProxyGroup{}
		}

		return tx.Commit()
	}()
	if err != nil {
		return ClashConfigSubscription{}, err
	}

	return sub, nil
}

func (r *Repository) loadClashConfigProviders(ctx context.Context, subID int64) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT provider_id FROM clash_config_subscription_providers WHERE subscription_id = ? ORDER BY position`, subID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if ids == nil {
		ids = []int64{}
	}
	return ids, rows.Err()
}

func (r *Repository) loadClashConfigProxyGroups(ctx context.Context, subID int64) ([]ClashConfigProxyGroup, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, type, position, url, interval_seconds, bind_internal_proxy_group_id, is_system FROM clash_config_proxy_groups WHERE subscription_id = ? ORDER BY position, id`, subID,
	)
	if err != nil {
		return nil, err
	}

	var groups []ClashConfigProxyGroup
	for rows.Next() {
		var g ClashConfigProxyGroup
		var isSystem int64
		var bindID sql.NullInt64
		if err := rows.Scan(&g.ID, &g.Name, &g.Type, &g.Position, &g.URL, &g.Interval, &bindID, &isSystem); err != nil {
			rows.Close()
			return nil, err
		}
		g.BindInternalGroupID = bindID.Int64
		g.IsSystem = isSystem == 1
		groups = append(groups, g)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	for i := range groups {
		members, err := r.loadProxyGroupMembers(ctx, groups[i].ID)
		if err != nil {
			return nil, err
		}
		groups[i].Proxies = members
	}

	if groups == nil {
		groups = []ClashConfigProxyGroup{}
	}
	return groups, nil
}

func (r *Repository) loadProxyGroupMembers(ctx context.Context, pgID int64) ([]ProxyMember, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT member_type, member_value FROM clash_config_proxy_group_members WHERE proxy_group_id = ? ORDER BY position`, pgID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []ProxyMember
	for rows.Next() {
		var m ProxyMember
		if err := rows.Scan(&m.Type, &m.Value); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	if members == nil {
		members = []ProxyMember{}
	}
	return members, rows.Err()
}

func (r *Repository) UpdateClashConfig(ctx context.Context, id int64, in UpdateClashConfigSubscriptionInput) (ClashConfigSubscription, error) {
	err := func() error {
		tx, err := r.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()

		nowStr := nowInLocation().Format(time.RFC3339)

		_, err = tx.ExecContext(ctx,
			`UPDATE clash_config_subscriptions SET name = ?, updated_at = ? WHERE id = ?`,
			in.Name, nowStr, id,
		)
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx, `DELETE FROM clash_config_proxy_group_members WHERE proxy_group_id IN (SELECT id FROM clash_config_proxy_groups WHERE subscription_id = ?)`, id)
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx, `DELETE FROM clash_config_proxy_groups WHERE subscription_id = ?`, id)
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx, `DELETE FROM clash_config_subscription_providers WHERE subscription_id = ?`, id)
		if err != nil {
			return err
		}

		for i, pid := range in.Providers {
			_, err = tx.ExecContext(ctx,
				`INSERT INTO clash_config_subscription_providers (subscription_id, provider_id, position) VALUES (?, ?, ?)`,
				id, pid, i,
			)
			if err != nil {
				return err
			}
		}

		for i, pg := range in.ProxyGroups {
			pg.Position = int64(i)
			_, err = insertProxyGroup(ctx, tx, id, pg, nowStr)
			if err != nil {
				return err
			}
		}

		if err := tx.Commit(); err != nil {
			return err
		}
		return nil
	}()
	if err != nil {
		return ClashConfigSubscription{}, err
	}

	return r.GetClashConfigByID(ctx, id)
}

func (r *Repository) DeleteClashConfig(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM clash_config_subscriptions WHERE id = ?`, id)
	return err
}

func (r *Repository) CreateProxyProvider(ctx context.Context, in CreateProxyProviderSubscriptionInput) (ProxyProviderSubscription, error) {
	var sub ProxyProviderSubscription

	err := func() error {
		tx, err := r.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()

		now := nowInLocation()
		nowStr := now.Format(time.RFC3339)

		result, err := tx.ExecContext(ctx,
			`INSERT INTO proxy_provider_subscriptions (name, internal_proxy_group_id, created_at, updated_at) VALUES (?, ?, ?, ?)`,
			in.Name, in.InternalProxyGroupID, nowStr, nowStr,
		)
		if err != nil {
			return err
		}
		subID, _ := result.LastInsertId()

		for i, pid := range in.Providers {
			_, err = tx.ExecContext(ctx,
				`INSERT INTO proxy_provider_subscription_providers (subscription_id, provider_id, position) VALUES (?, ?, ?)`,
				subID, pid, i,
			)
			if err != nil {
				return err
			}
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		sub = ProxyProviderSubscription{
			ID:                   subID,
			Name:                 in.Name,
			Providers:            in.Providers,
			InternalProxyGroupID: in.InternalProxyGroupID,
			CreatedAt:            now,
			UpdatedAt:            now,
		}
		return nil
	}()
	if err != nil {
		return ProxyProviderSubscription{}, err
	}

	return sub, nil
}

func (r *Repository) GetProxyProviderByID(ctx context.Context, id int64) (ProxyProviderSubscription, error) {
	var sub ProxyProviderSubscription
	var createdAt, updatedAt string
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, internal_proxy_group_id, created_at, updated_at FROM proxy_provider_subscriptions WHERE id = ?`, id,
	).Scan(&sub.ID, &sub.Name, &sub.InternalProxyGroupID, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ProxyProviderSubscription{}, ErrNotFound
		}
		return ProxyProviderSubscription{}, err
	}
	sub.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	sub.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	sub.Providers, err = r.loadSubscriptionProviders(ctx, "proxy_provider_subscription_providers", id)
	if err != nil {
		return ProxyProviderSubscription{}, err
	}

	return sub, nil
}

func (r *Repository) UpdateProxyProvider(ctx context.Context, id int64, in UpdateProxyProviderSubscriptionInput) (ProxyProviderSubscription, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ProxyProviderSubscription{}, err
	}
	defer tx.Rollback()

	now := nowInLocation()
	nowStr := now.Format(time.RFC3339)

	_, err = tx.ExecContext(ctx,
		`UPDATE proxy_provider_subscriptions SET name = ?, internal_proxy_group_id = ?, updated_at = ? WHERE id = ?`,
		in.Name, in.InternalProxyGroupID, nowStr, id,
	)
	if err != nil {
		return ProxyProviderSubscription{}, err
	}

	_, err = tx.ExecContext(ctx, `DELETE FROM proxy_provider_subscription_providers WHERE subscription_id = ?`, id)
	if err != nil {
		return ProxyProviderSubscription{}, err
	}

	for i, pid := range in.Providers {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO proxy_provider_subscription_providers (subscription_id, provider_id, position) VALUES (?, ?, ?)`,
			id, pid, i,
		)
		if err != nil {
			return ProxyProviderSubscription{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return ProxyProviderSubscription{}, err
	}

	return r.GetProxyProviderByID(ctx, id)
}

func (r *Repository) DeleteProxyProvider(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM proxy_provider_subscriptions WHERE id = ?`, id)
	return err
}

func (r *Repository) CreateRuleProvider(ctx context.Context, in CreateRuleProviderSubscriptionInput) (RuleProviderSubscription, error) {
	var sub RuleProviderSubscription

	err := func() error {
		tx, err := r.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()

		now := nowInLocation()
		nowStr := now.Format(time.RFC3339)

		result, err := tx.ExecContext(ctx,
			`INSERT INTO rule_provider_subscriptions (name, internal_proxy_group_id, created_at, updated_at) VALUES (?, ?, ?, ?)`,
			in.Name, in.InternalProxyGroupID, nowStr, nowStr,
		)
		if err != nil {
			return err
		}
		subID, _ := result.LastInsertId()

		for i, pid := range in.Providers {
			_, err = tx.ExecContext(ctx,
				`INSERT INTO rule_provider_subscription_providers (subscription_id, provider_id, position) VALUES (?, ?, ?)`,
				subID, pid, i,
			)
			if err != nil {
				return err
			}
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		sub = RuleProviderSubscription{
			ID:                   subID,
			Name:                 in.Name,
			Providers:            in.Providers,
			InternalProxyGroupID: in.InternalProxyGroupID,
			CreatedAt:            now,
			UpdatedAt:            now,
		}
		return nil
	}()
	if err != nil {
		return RuleProviderSubscription{}, err
	}

	return sub, nil
}

func (r *Repository) GetRuleProviderByID(ctx context.Context, id int64) (RuleProviderSubscription, error) {
	var sub RuleProviderSubscription
	var createdAt, updatedAt string
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, internal_proxy_group_id, created_at, updated_at FROM rule_provider_subscriptions WHERE id = ?`, id,
	).Scan(&sub.ID, &sub.Name, &sub.InternalProxyGroupID, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return RuleProviderSubscription{}, ErrNotFound
		}
		return RuleProviderSubscription{}, err
	}
	sub.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	sub.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	sub.Providers, err = r.loadSubscriptionProviders(ctx, "rule_provider_subscription_providers", id)
	if err != nil {
		return RuleProviderSubscription{}, err
	}

	return sub, nil
}

func (r *Repository) UpdateRuleProvider(ctx context.Context, id int64, in UpdateRuleProviderSubscriptionInput) (RuleProviderSubscription, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return RuleProviderSubscription{}, err
	}
	defer tx.Rollback()

	now := nowInLocation()
	nowStr := now.Format(time.RFC3339)

	_, err = tx.ExecContext(ctx,
		`UPDATE rule_provider_subscriptions SET name = ?, internal_proxy_group_id = ?, updated_at = ? WHERE id = ?`,
		in.Name, in.InternalProxyGroupID, nowStr, id,
	)
	if err != nil {
		return RuleProviderSubscription{}, err
	}

	_, err = tx.ExecContext(ctx, `DELETE FROM rule_provider_subscription_providers WHERE subscription_id = ?`, id)
	if err != nil {
		return RuleProviderSubscription{}, err
	}

	for i, pid := range in.Providers {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO rule_provider_subscription_providers (subscription_id, provider_id, position) VALUES (?, ?, ?)`,
			id, pid, i,
		)
		if err != nil {
			return RuleProviderSubscription{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return RuleProviderSubscription{}, err
	}

	return r.GetRuleProviderByID(ctx, id)
}

func (r *Repository) DeleteRuleProvider(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM rule_provider_subscriptions WHERE id = ?`, id)
	return err
}

func (r *Repository) loadSubscriptionProviders(ctx context.Context, table string, subID int64) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT provider_id FROM `+table+` WHERE subscription_id = ? ORDER BY position`, subID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if ids == nil {
		ids = []int64{}
	}
	return ids, rows.Err()
}

func (r *Repository) ProviderReferencedByAnySubscription(ctx context.Context, providerID int64) (bool, error) {
	var count int64

	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM clash_config_subscription_providers WHERE provider_id = ?`, providerID,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}

	err = r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM proxy_provider_subscription_providers WHERE provider_id = ?`, providerID,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}

	err = r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM rule_provider_subscription_providers WHERE provider_id = ?`, providerID,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}

	return false, nil
}
