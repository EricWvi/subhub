# AGENTS.md

## Commands

```bash
go test ./...                    # run all tests
go test -race ./...              # run all tests with race detector
go test ./tests/integration -run TestName -v   # single integration test
go test ./tests/unit -run TestName -v          # single unit test
go vet ./...                     # static analysis
go build .                       # build the binary
```

There is no lint or typecheck command beyond `go vet`.

## Architecture

Single Go HTTP service. Entry point: `cmd/subhub/main.go` wires everything:

```
config.Load → store.MustOpen (SQLite) → provider.Repository → provider.Service → provider.Handler
                                                            ↘ refresh.Service → refresh.Scheduler (background)
                                                            ↘ output.Handler → render.MihomoTemplate
```

**Package boundaries** (all under `internal/`):

- `config` — hardcoded defaults (no env/flag parsing)
- `store` — SQLite open + embedded SQL migrations (`store/migrations/`)
- `provider` — domain models, repository (SQL CRUD), service (validation/business rules), HTTP handlers
- `fetch` — upstream HTTP client with timeout
- `parse` — YAML and Base64 subscription decoding → `[]map[string]any` proxy maps
- `refresh` — refresh pipeline (fetch→parse→snapshot) and background scheduler
- `render` — Mihomo YAML template injection
- `output` — `GET /subscriptions/mihomo` handler

**Data flow:** Upstream subscription → fetch → parse → provider_snapshots (SQLite) → render from last-known-good → Mihomo YAML output.

## Key conventions

- **No code comments.** Do not add comments unless explicitly asked.
- **SQLite driver:** `modernc.org/sqlite` (pure Go, not `mattn/go-sqlite3`). Driver name is `"sqlite"`. Connection pool is single-writer (`MaxOpenConns=1`).
- **Migrations** are embedded via `//go:embed migrations/*.sql` and use `CREATE TABLE IF NOT EXISTS`.
- **Provider handler routing:** `provider/http.go` manually parses `/providers/{id}` and `/providers/{id}/refresh` from URL path via `strings.TrimPrefix` + `strings.Split` (no router library).
- **Refresh injection:** `provider.Handler.SetRefresher(RefreshProviderFunc)` avoids circular deps between `provider` and `refresh` packages.
- **Test proxy maps** are `[]map[string]any`, not structs — this is the Mihomo-native format throughout.
- All struct definitions must include `json` tags.
- all API paths should start with `/api`
- use `Asia/Shanghai` as Timezone
- use 24h formats

## Testing

- Integration tests: `tests/integration/` (package `integration`) — use `httptest.NewServer` + real SQLite in `t.TempDir()`
- Unit tests: `tests/unit/` — parser and renderer tests with fixtures
- Fixtures: `tests/fixtures/` — `template.yaml`, `provider_plain.yaml`, `provider_base64.txt`
- All test helpers (`newTestServer`, `postJSON`, `readBody`, etc.) live in the integration test files themselves
- `stretchr/testify` for assertions (`assert`/`require`)

## Gotchas

- `config.DatabasePath` defaults to `"data/subhub.db"` (relative path). Tests override this to `t.TempDir()`.
- The template path for Mihomo output is hardcoded to `"tests/fixtures/template.yaml"` in `main.go`.
- `store.MustOpen` calls `log.Fatalf` on failure (not test-friendly for direct use outside `main`). Tests call it in `newTestServer` where `log.Fatalf` is acceptable since `t.TempDir()` won't fail.
