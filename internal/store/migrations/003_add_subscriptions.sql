CREATE TABLE IF NOT EXISTS clash_config_subscriptions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS clash_config_subscription_providers (
    subscription_id INTEGER NOT NULL,
    provider_id INTEGER NOT NULL,
    position INTEGER NOT NULL,
    PRIMARY KEY (subscription_id, position),
    UNIQUE (subscription_id, provider_id),
    FOREIGN KEY(subscription_id) REFERENCES clash_config_subscriptions(id) ON DELETE CASCADE,
    FOREIGN KEY(provider_id) REFERENCES providers(id)
);

CREATE TABLE IF NOT EXISTS clash_config_proxy_groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    subscription_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    url TEXT NOT NULL DEFAULT '',
    interval_seconds INTEGER NOT NULL DEFAULT 0,
    bind_internal_proxy_group_id INTEGER,
    is_system INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (subscription_id, name),
    UNIQUE (subscription_id, bind_internal_proxy_group_id),
    FOREIGN KEY(subscription_id) REFERENCES clash_config_subscriptions(id) ON DELETE CASCADE,
    FOREIGN KEY(bind_internal_proxy_group_id) REFERENCES proxy_groups(id)
);

CREATE TABLE IF NOT EXISTS clash_config_proxy_group_members (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    proxy_group_id INTEGER NOT NULL,
    position INTEGER NOT NULL,
    member_type TEXT NOT NULL,
    member_value TEXT NOT NULL,
    UNIQUE (proxy_group_id, position),
    FOREIGN KEY(proxy_group_id) REFERENCES clash_config_proxy_groups(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS proxy_provider_subscriptions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    internal_proxy_group_id INTEGER NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY(internal_proxy_group_id) REFERENCES proxy_groups(id)
);

CREATE TABLE IF NOT EXISTS proxy_provider_subscription_providers (
    subscription_id INTEGER NOT NULL,
    provider_id INTEGER NOT NULL,
    position INTEGER NOT NULL,
    PRIMARY KEY (subscription_id, position),
    UNIQUE (subscription_id, provider_id),
    FOREIGN KEY(subscription_id) REFERENCES proxy_provider_subscriptions(id) ON DELETE CASCADE,
    FOREIGN KEY(provider_id) REFERENCES providers(id)
);

CREATE TABLE IF NOT EXISTS rule_provider_subscriptions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    internal_proxy_group_id INTEGER NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY(internal_proxy_group_id) REFERENCES proxy_groups(id)
);

CREATE TABLE IF NOT EXISTS rule_provider_subscription_providers (
    subscription_id INTEGER NOT NULL,
    provider_id INTEGER NOT NULL,
    position INTEGER NOT NULL,
    PRIMARY KEY (subscription_id, position),
    UNIQUE (subscription_id, provider_id),
    FOREIGN KEY(subscription_id) REFERENCES rule_provider_subscriptions(id) ON DELETE CASCADE,
    FOREIGN KEY(provider_id) REFERENCES providers(id)
);
