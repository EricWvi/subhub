package subscription

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

func (r *Repository) ListNodeProviders(ctx context.Context) ([]NodeProviderSubscription, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, created_at, updated_at FROM node_provider_subscriptions ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	subs := []NodeProviderSubscription{}
	for rows.Next() {
		var sub NodeProviderSubscription
		var createdAt, updatedAt string
		if err := rows.Scan(&sub.ID, &sub.Name, &createdAt, &updatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		sub.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		sub.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		subs = append(subs, sub)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	for i := range subs {
		subs[i].Nodes, err = r.loadNodeProviderNodes(ctx, subs[i].ID)
		if err != nil {
			return nil, err
		}
	}
	return subs, nil
}

func (r *Repository) loadNodeProviderNodes(ctx context.Context, id int64) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT node_id FROM node_provider_subscription_nodes WHERE subscription_id = ? ORDER BY position`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	nodes := []int64{}
	for rows.Next() {
		var nodeID int64
		if err := rows.Scan(&nodeID); err != nil {
			return nil, err
		}
		nodes = append(nodes, nodeID)
	}
	return nodes, rows.Err()
}

func (r *Repository) GetNodeProviderByID(ctx context.Context, id int64) (NodeProviderSubscription, error) {
	var sub NodeProviderSubscription
	var createdAt, updatedAt string
	err := r.db.QueryRowContext(ctx, `SELECT id, name, created_at, updated_at FROM node_provider_subscriptions WHERE id = ?`, id).Scan(&sub.ID, &sub.Name, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return sub, ErrNotFound
	}
	if err != nil {
		return sub, err
	}
	sub.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	sub.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	sub.Nodes, err = r.loadNodeProviderNodes(ctx, id)
	return sub, err
}

func (r *Repository) CreateNodeProvider(ctx context.Context, in CreateNodeProviderSubscriptionInput) (NodeProviderSubscription, error) {
	return r.saveNodeProvider(ctx, 0, in)
}

func (r *Repository) UpdateNodeProvider(ctx context.Context, id int64, in UpdateNodeProviderSubscriptionInput) (NodeProviderSubscription, error) {
	return r.saveNodeProvider(ctx, id, in)
}

func (r *Repository) saveNodeProvider(ctx context.Context, id int64, in CreateNodeProviderSubscriptionInput) (NodeProviderSubscription, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return NodeProviderSubscription{}, err
	}
	defer tx.Rollback()
	now := nowInLocation().Format(time.RFC3339)
	if id == 0 {
		result, err := tx.ExecContext(ctx, `INSERT INTO node_provider_subscriptions (name, created_at, updated_at) VALUES (?, ?, ?)`, in.Name, now, now)
		if err != nil {
			return NodeProviderSubscription{}, err
		}
		id, err = result.LastInsertId()
		if err != nil {
			return NodeProviderSubscription{}, err
		}
	} else {
		result, err := tx.ExecContext(ctx, `UPDATE node_provider_subscriptions SET name = ?, updated_at = ? WHERE id = ?`, in.Name, now, id)
		if err != nil {
			return NodeProviderSubscription{}, err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return NodeProviderSubscription{}, err
		}
		if affected == 0 {
			return NodeProviderSubscription{}, ErrNotFound
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM node_provider_subscription_nodes WHERE subscription_id = ?`, id); err != nil {
			return NodeProviderSubscription{}, err
		}
	}
	for i, nodeID := range in.Nodes {
		if _, err := tx.ExecContext(ctx, `INSERT INTO node_provider_subscription_nodes (subscription_id, node_id, position) VALUES (?, ?, ?)`, id, nodeID, i); err != nil {
			return NodeProviderSubscription{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return NodeProviderSubscription{}, err
	}
	return r.GetNodeProviderByID(ctx, id)
}

func (r *Repository) DeleteNodeProvider(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM node_provider_subscriptions WHERE id = ?`, id)
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
