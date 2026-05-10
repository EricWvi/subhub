CREATE TABLE IF NOT EXISTS rule_provider_subscriptions_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    internal_proxy_group_id INTEGER NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

INSERT INTO rule_provider_subscriptions_new SELECT id, name, internal_proxy_group_id, created_at, updated_at FROM rule_provider_subscriptions;

DROP TABLE rule_provider_subscriptions;

ALTER TABLE rule_provider_subscriptions_new RENAME TO rule_provider_subscriptions;
