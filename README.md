# SubHub

A subscription aggregation service for Clash/mihomo proxies. Fetches content from multiple airport URLs, caches results, and outputs valid Clash YAML configuration.

## Phase 1 API

- `GET /providers`
- `POST /providers`
- `PUT /providers/{id}`
- `DELETE /providers/{id}`
- `POST /providers/{id}/refresh`
- `GET /subscriptions/mihomo`

New providers default to a 2 hour refresh interval when `refresh_interval_seconds` is omitted.
Supported upstream payload styles in Phase 1 are plain YAML and Base64-encoded YAML containing a `proxies` list.
