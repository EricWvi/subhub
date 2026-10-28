CREATE TABLE IF NOT EXISTS node_provider_subscriptions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS node_provider_subscription_nodes (
    subscription_id INTEGER NOT NULL REFERENCES node_provider_subscriptions(id) ON DELETE CASCADE,
    node_id INTEGER NOT NULL REFERENCES custom_nodes(id) ON DELETE CASCADE,
    position INTEGER NOT NULL,
    PRIMARY KEY (subscription_id, node_id)
);
