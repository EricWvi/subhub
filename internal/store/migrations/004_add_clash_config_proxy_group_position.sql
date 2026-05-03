ALTER TABLE clash_config_proxy_groups ADD COLUMN position INTEGER NOT NULL DEFAULT 0;

WITH ordered AS (
    SELECT id, ROW_NUMBER() OVER (
        PARTITION BY subscription_id
        ORDER BY id
    ) - 1 AS new_position
    FROM clash_config_proxy_groups
)
UPDATE clash_config_proxy_groups
SET position = (
    SELECT new_position
    FROM ordered
    WHERE ordered.id = clash_config_proxy_groups.id
);
