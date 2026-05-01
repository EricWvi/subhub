# Phase 1 Core Engine Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Phase 1 SubHub foundation: SQLite-backed provider management, scheduled refresh with last-known-good retention, normalization of YAML and Base64 subscription payloads into Mihomo-native node maps, and a unified Mihomo output endpoint.

**Architecture:** Use a small Go HTTP service with clear package boundaries: `provider` for CRUD and refresh policy, `fetch` for HTTP retrieval and content decoding, `parse` for normalization into Mihomo-native `[]map[string]any` proxy payloads, `store` for SQLite persistence, and `render` for Mihomo template output. Persist provider definitions, refresh schedules, fetch attempts, and provider snapshots in SQLite so the system can survive upstream failures and always render from the most recent usable snapshot.

**Tech Stack:** Go 1.26, `database/sql` with `modernc.org/sqlite`, `net/http`, `gopkg.in/yaml.v3`, standard-library `encoding/base64`, `httptest`, and table-driven tests.

---

## File Structure

- `cmd/subhub/main.go`
  Entry point that wires config, SQLite, HTTP routes, and the refresh scheduler.
- `internal/config/config.go`
  Runtime configuration defaults, including database path, listen address, request timeout, and the default refresh interval of 2 hours.
- `internal/store/sqlite.go`
  SQLite connection setup, schema migration runner, and transaction helpers.
- `internal/store/migrations/001_initial.sql`
  Initial schema for providers, provider snapshots, and refresh attempts.
- `internal/provider/model.go`
  Provider and snapshot domain models.
- `internal/provider/repository.go`
  CRUD and lookup methods for providers and last-known-good snapshots.
- `internal/provider/service.go`
  Business rules for create/update/delete/list/get provider operations and default interval handling.
- `internal/provider/http.go`
  HTTP handlers for provider CRUD and manual refresh.
- `internal/fetch/client.go`
  Fetch upstream provider payloads with timeout and status validation.
- `internal/parse/subscription.go`
  YAML and Base64 payload decoding plus normalization into Mihomo-native proxy maps.
- `internal/refresh/service.go`
  Refresh workflow orchestration: fetch, decode, parse, persist snapshot, record failure without deleting good data.
- `internal/refresh/scheduler.go`
  Background scheduler that refreshes providers on their own interval and defaults new providers to 2 hours.
- `internal/render/mihomo.go`
  Merge normalized nodes into the static Mihomo template fixture.
- `internal/output/http.go`
  HTTP handler for unified Mihomo output generation.
- `tests/fixtures/template.yaml`
  Static Mihomo template fixture already present in the repo.
- `tests/fixtures/provider_plain.yaml`
  Plain YAML subscription fixture.
- `tests/fixtures/provider_base64.txt`
  Base64-encoded subscription fixture.
- `tests/integration/providers_api_test.go`
  Provider CRUD and manual refresh endpoint coverage.
- `tests/integration/refresh_pipeline_test.go`
  End-to-end refresh, retention, and rendering coverage.
- `tests/unit/parse_subscription_test.go`
  Parser tests for YAML and Base64 inputs.
- `tests/unit/render_mihomo_test.go`
  Mihomo output rendering tests.

### Task 1: Bootstrap the Service Skeleton and SQLite Schema

**Files:**
- Create: `cmd/subhub/main.go`
- Create: `internal/config/config.go`
- Create: `internal/store/sqlite.go`
- Create: `internal/store/migrations/001_initial.sql`
- Create: `internal/provider/model.go`
- Test: `tests/integration/providers_api_test.go`

- [ ] **Step 1: Write the failing integration test for service boot**

```go
func TestListProvidersStartsEmpty(t *testing.T) {
	ts := newTestServer(t)

	resp, err := http.Get(ts.URL + "/providers")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.JSONEq(t, `{"providers":[]}`, readBody(t, resp))
}
```

- [ ] **Step 2: Run the test to verify the server is not implemented yet**

Run: `go test ./tests/integration -run TestListProvidersStartsEmpty -v`
Expected: FAIL with a compile error such as `undefined: newTestServer` or missing package symbols.

- [ ] **Step 3: Add config defaults, SQLite bootstrap, and initial schema**

```sql
CREATE TABLE providers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    url TEXT NOT NULL,
    refresh_interval_seconds INTEGER NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE provider_snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    provider_id INTEGER NOT NULL,
    format TEXT NOT NULL,
    raw_payload BLOB NOT NULL,
    normalized_yaml TEXT NOT NULL,
    node_count INTEGER NOT NULL,
    fetched_at TEXT NOT NULL,
    is_last_known_good INTEGER NOT NULL DEFAULT 1,
    FOREIGN KEY(provider_id) REFERENCES providers(id) ON DELETE CASCADE
);

CREATE TABLE refresh_attempts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    provider_id INTEGER NOT NULL,
    status TEXT NOT NULL,
    message TEXT NOT NULL,
    attempted_at TEXT NOT NULL,
    FOREIGN KEY(provider_id) REFERENCES providers(id) ON DELETE CASCADE
);
```

```go
const DefaultRefreshInterval = 2 * time.Hour

type Config struct {
	ListenAddr             string
	DatabasePath           string
	UpstreamRequestTimeout time.Duration
	DefaultRefreshInterval time.Duration
}

func Load() Config {
	return Config{
		ListenAddr:             ":8080",
		DatabasePath:           "data/subhub.db",
		UpstreamRequestTimeout: 15 * time.Second,
		DefaultRefreshInterval: DefaultRefreshInterval,
	}
}
```

- [ ] **Step 4: Add the minimal HTTP server that returns an empty provider list**

```go
func main() {
	cfg := config.Load()
	db := store.MustOpen(cfg.DatabasePath)
	defer db.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/providers", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"providers":[]}`)
	})

	log.Fatal(http.ListenAndServe(cfg.ListenAddr, mux))
}
```

- [ ] **Step 5: Run the integration test and verify it passes**

Run: `go test ./tests/integration -run TestListProvidersStartsEmpty -v`
Expected: PASS

- [ ] **Step 6: Commit the bootstrap**

```bash
git add cmd/subhub/main.go internal/config/config.go internal/store/sqlite.go internal/store/migrations/001_initial.sql internal/provider/model.go tests/integration/providers_api_test.go
git commit -m "feat: bootstrap subhub service and sqlite schema"
```

### Task 2: Implement Provider CRUD with a 2-Hour Default Refresh Interval

**Files:**
- Modify: `cmd/subhub/main.go`
- Create: `internal/provider/repository.go`
- Create: `internal/provider/service.go`
- Create: `internal/provider/http.go`
- Modify: `tests/integration/providers_api_test.go`

- [ ] **Step 1: Write failing tests for create, list, update, and delete provider flows**

```go
func TestCreateProviderUsesDefaultRefreshInterval(t *testing.T) {
	ts := newTestServer(t)

	body := `{"name":"alpha","url":"https://example.com/sub"}`
	resp := postJSON(t, ts.URL+"/providers", body)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.JSONEq(t, `{
	  "provider":{
	    "name":"alpha",
	    "url":"https://example.com/sub",
	    "refresh_interval_seconds":7200
	  }
	}`, readBody(t, resp))
}

func TestUpdateAndDeleteProvider(t *testing.T) {
	ts := newTestServer(t)
	id := createProvider(t, ts.URL, `{"name":"alpha","url":"https://example.com/sub"}`)

	putResp := putJSON(t, fmt.Sprintf("%s/providers/%d", ts.URL, id), `{"name":"beta","url":"https://example.net/sub","refresh_interval_seconds":3600}`)
	assert.Equal(t, http.StatusOK, putResp.StatusCode)

	delResp := deleteRequest(t, fmt.Sprintf("%s/providers/%d", ts.URL, id))
	assert.Equal(t, http.StatusNoContent, delResp.StatusCode)
}
```

- [ ] **Step 2: Run the CRUD tests to verify they fail**

Run: `go test ./tests/integration -run 'TestCreateProviderUsesDefaultRefreshInterval|TestUpdateAndDeleteProvider' -v`
Expected: FAIL with `404` responses or missing route handlers.

- [ ] **Step 3: Implement repository methods and provider service rules**

```go
type Provider struct {
	ID                     int64     `json:"id"`
	Name                   string    `json:"name"`
	URL                    string    `json:"url"`
	RefreshIntervalSeconds int64     `json:"refresh_interval_seconds"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

func (s *Service) Create(ctx context.Context, in CreateProviderInput) (Provider, error) {
	interval := in.RefreshIntervalSeconds
	if interval == 0 {
		interval = int64((2 * time.Hour).Seconds())
	}
	if interval < 300 {
		return Provider{}, ErrRefreshIntervalTooShort
	}
	return s.repo.Create(ctx, Provider{
		Name:                   in.Name,
		URL:                    in.URL,
		RefreshIntervalSeconds: interval,
	})
}
```

- [ ] **Step 4: Add HTTP handlers for `GET/POST /providers` and `PUT/DELETE /providers/{id}`**

```go
func (h *Handler) routes(mux *http.ServeMux) {
	mux.HandleFunc("/providers", h.handleProviders)
	mux.HandleFunc("/providers/", h.handleProviderByID)
}
```

```go
switch r.Method {
case http.MethodGet:
	h.listProviders(w, r)
case http.MethodPost:
	h.createProvider(w, r)
default:
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}
```

- [ ] **Step 5: Run the CRUD integration tests and full package tests**

Run: `go test ./tests/integration -run 'TestCreateProviderUsesDefaultRefreshInterval|TestUpdateAndDeleteProvider' -v`
Expected: PASS

Run: `go test ./...`
Expected: PASS

- [ ] **Step 6: Commit the provider CRUD slice**

```bash
git add cmd/subhub/main.go internal/provider/repository.go internal/provider/service.go internal/provider/http.go tests/integration/providers_api_test.go
git commit -m "feat: add provider crud endpoints"
```

### Task 3: Add YAML and Base64 Subscription Parsing into Mihomo-Native Proxy Maps

**Files:**
- Create: `internal/parse/subscription.go`
- Create: `tests/fixtures/provider_plain.yaml`
- Create: `tests/fixtures/provider_base64.txt`
- Create: `tests/unit/parse_subscription_test.go`

- [ ] **Step 1: Write failing parser tests for plain YAML and Base64 input**

```go
func TestParseSubscriptionAcceptsPlainYAML(t *testing.T) {
	payload := fixtureBytes(t, "tests/fixtures/provider_plain.yaml")

	nodes, format, err := parse.DecodeAndNormalize(payload)
	require.NoError(t, err)
	assert.Equal(t, "yaml", format)
	assert.Len(t, nodes, 2)
	assert.Equal(t, "vmess-hk-01", nodes[0].Name)
}

func TestParseSubscriptionAcceptsBase64EncodedYAML(t *testing.T) {
	payload := fixtureBytes(t, "tests/fixtures/provider_base64.txt")

	nodes, format, err := parse.DecodeAndNormalize(payload)
	require.NoError(t, err)
	assert.Equal(t, "base64+yaml", format)
	assert.Len(t, nodes, 2)
}
```

- [ ] **Step 2: Run the parser tests to verify they fail first**

Run: `go test ./tests/unit -run 'TestParseSubscriptionAcceptsPlainYAML|TestParseSubscriptionAcceptsBase64EncodedYAML' -v`
Expected: FAIL because the `parse` package and fixtures do not exist yet.

- [ ] **Step 3: Add the decode pipeline that returns Mihomo-compatible proxy maps**

```go
type ProxySchema struct {
	Proxies []map[string]any `yaml:"proxies"`
}

func DecodeAndNormalize(payload []byte) ([]map[string]any, string, error) {
	trimmed := bytes.TrimSpace(payload)
	if decoded, err := tryBase64(trimmed); err == nil {
		return parseYAML(decoded, "base64+yaml")
	}
	return parseYAML(trimmed, "yaml")
}
```

- [ ] **Step 4: Seed fixtures that represent the supported Phase 1 formats**

```yaml
proxies:
  - name: vmess-hk-01
    type: vmess
    server: hk.example.com
    port: 443
    uuid: 11111111-1111-1111-1111-111111111111
    cipher: auto
    udp: true
  - name: ss-jp-01
    type: ss
    server: jp.example.com
    port: 8388
    cipher: aes-128-gcm
    password: secret
```

- [ ] **Step 5: Run parser tests and then the whole suite**

Run: `go test ./tests/unit -run 'TestParseSubscriptionAcceptsPlainYAML|TestParseSubscriptionAcceptsBase64EncodedYAML' -v`
Expected: PASS

Run: `go test ./...`
Expected: PASS

- [ ] **Step 6: Commit parser support**

```bash
git add internal/parse/subscription.go tests/fixtures/provider_plain.yaml tests/fixtures/provider_base64.txt tests/unit/parse_subscription_test.go
git commit -m "feat: add yaml and base64 subscription parsing"
```

### Task 4: Build the Refresh Pipeline with Snapshot Retention and Failure Recording

**Files:**
- Create: `internal/fetch/client.go`
- Create: `internal/refresh/service.go`
- Modify: `internal/provider/repository.go`
- Create: `tests/integration/refresh_pipeline_test.go`

- [ ] **Step 1: Write a failing end-to-end refresh test that proves last-known-good retention**

```go
func TestRefreshRetainsPreviousSnapshotWhenProviderFails(t *testing.T) {
	upstream := newFlakyProviderServer(t)
	ts := newTestServerWithUpstream(t, upstream)
	providerID := createProvider(t, ts.URL, fmt.Sprintf(`{"name":"alpha","url":"%s"}`, upstream.SuccessURL()))

	refreshProvider(t, ts.URL, providerID)
	firstOutput := getText(t, ts.URL+"/subscriptions/mihomo")
	require.Contains(t, firstOutput, "vmess-hk-01")

	upstream.FailNext(http.StatusBadGateway, "upstream down")
	refreshResp := postJSON(t, fmt.Sprintf("%s/providers/%d/refresh", ts.URL, providerID), `{}`)
	assert.Equal(t, http.StatusBadGateway, refreshResp.StatusCode)

	secondOutput := getText(t, ts.URL+"/subscriptions/mihomo")
	assert.Equal(t, firstOutput, secondOutput)
}
```

- [ ] **Step 2: Run the refresh retention test to confirm the pipeline is missing**

Run: `go test ./tests/integration -run TestRefreshRetainsPreviousSnapshotWhenProviderFails -v`
Expected: FAIL because the refresh route and renderer do not exist.

- [ ] **Step 3: Implement fetch, parse, and snapshot persistence in one refresh transaction**

```go
func (s *Service) RefreshProvider(ctx context.Context, providerID int64) error {
	provider, err := s.providers.Get(ctx, providerID)
	if err != nil {
		return err
	}

	payload, err := s.fetcher.Fetch(ctx, provider.URL)
	if err != nil {
		return s.providers.RecordRefreshFailure(ctx, providerID, err.Error())
	}

	nodes, format, err := s.parser.DecodeAndNormalize(payload)
	if err != nil {
		return s.providers.RecordRefreshFailure(ctx, providerID, err.Error())
	}

	return s.providers.ReplaceLastKnownGoodSnapshot(ctx, providerID, format, payload, nodes)
}
```

- [ ] **Step 4: Add `POST /providers/{id}/refresh` and ensure failures do not delete prior snapshots**

```go
func (r *Repository) ReplaceLastKnownGoodSnapshot(ctx context.Context, providerID int64, format string, raw []byte, nodes []map[string]any) error {
	normalizedYAML, err := yaml.Marshal(map[string]any{"proxies": nodes})
	if err != nil {
		return err
	}

	return r.withTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `UPDATE provider_snapshots SET is_last_known_good = 0 WHERE provider_id = ?`, providerID); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO provider_snapshots (provider_id, format, raw_payload, normalized_yaml, node_count, fetched_at, is_last_known_good) VALUES (?, ?, ?, ?, ?, ?, 1)`,
			providerID, format, raw, string(normalizedYAML), len(nodes), time.Now().UTC().Format(time.RFC3339))
		return err
	})
}
```

- [ ] **Step 5: Run the targeted refresh test and the full suite**

Run: `go test ./tests/integration -run TestRefreshRetainsPreviousSnapshotWhenProviderFails -v`
Expected: PASS

Run: `go test ./...`
Expected: PASS

- [ ] **Step 6: Commit refresh and retention support**

```bash
git add internal/fetch/client.go internal/refresh/service.go internal/provider/repository.go internal/provider/http.go tests/integration/refresh_pipeline_test.go
git commit -m "feat: add refresh pipeline with snapshot retention"
```

### Task 5: Add the 2-Hour Background Scheduler and Provider Selection Logic

**Files:**
- Create: `internal/refresh/scheduler.go`
- Modify: `cmd/subhub/main.go`
- Modify: `tests/integration/refresh_pipeline_test.go`

- [ ] **Step 1: Write a failing scheduler test that proves providers are selected by interval**

```go
func TestSchedulerRefreshesDueProvidersOnly(t *testing.T) {
	repo := newInMemoryProviderRepo(t)
	svc := newCountingRefreshService(repo)

	repo.SeedProvider(provider.Provider{ID: 1, Name: "due", RefreshIntervalSeconds: 300, UpdatedAt: time.Now().Add(-10 * time.Minute)})
	repo.SeedProvider(provider.Provider{ID: 2, Name: "fresh", RefreshIntervalSeconds: 7200, UpdatedAt: time.Now().Add(-30 * time.Minute)})

	scheduler := refresh.NewScheduler(repo, svc, time.Minute)
	scheduler.RunOnce(context.Background())

	assert.Equal(t, []int64{1}, svc.RefreshedProviderIDs())
}
```

- [ ] **Step 2: Run the scheduler test and verify the due-provider logic is absent**

Run: `go test ./tests/integration -run TestSchedulerRefreshesDueProvidersOnly -v`
Expected: FAIL because `refresh.NewScheduler` or `RunOnce` does not exist yet.

- [ ] **Step 3: Implement the scheduler with safe periodic execution**

```go
func (s *Scheduler) RunOnce(ctx context.Context) {
	providers, err := s.repo.List(ctx)
	if err != nil {
		s.logger.Printf("list providers: %v", err)
		return
	}

	now := s.clock.Now().UTC()
	for _, p := range providers {
		if now.Sub(p.UpdatedAt) < time.Duration(p.RefreshIntervalSeconds)*time.Second {
			continue
		}
		if err := s.refresh.RefreshProvider(ctx, p.ID); err != nil {
			s.logger.Printf("refresh provider %d: %v", p.ID, err)
		}
	}
}
```

- [ ] **Step 4: Start the scheduler from `main.go` with the same 2-hour default interval policy**

```go
refreshScheduler := refresh.NewScheduler(providerRepo, refreshService, time.Minute)
go refreshScheduler.Start(context.Background())
```

- [ ] **Step 5: Run scheduler tests and smoke the whole test suite**

Run: `go test ./tests/integration -run TestSchedulerRefreshesDueProvidersOnly -v`
Expected: PASS

Run: `go test ./...`
Expected: PASS

- [ ] **Step 6: Commit background refresh scheduling**

```bash
git add internal/refresh/scheduler.go cmd/subhub/main.go tests/integration/refresh_pipeline_test.go
git commit -m "feat: schedule provider refreshes"
```

### Task 6: Render the Unified Mihomo Output from the Latest Good Snapshots

**Files:**
- Create: `internal/render/mihomo.go`
- Create: `internal/output/http.go`
- Modify: `cmd/subhub/main.go`
- Create: `tests/unit/render_mihomo_test.go`
- Modify: `tests/integration/refresh_pipeline_test.go`

- [ ] **Step 1: Write failing renderer tests for merged multi-provider output**

```go
func TestRenderMihomoInjectsNormalizedNodesIntoTemplate(t *testing.T) {
	nodes := []map[string]any{
		{"name": "vmess-hk-01", "type": "vmess", "server": "hk.example.com", "port": 443},
		{"name": "ss-jp-01", "type": "ss", "server": "jp.example.com", "port": 8388},
	}

	out, err := render.MihomoTemplate("tests/fixtures/template.yaml", nodes)
	require.NoError(t, err)
	assert.Contains(t, out, "proxies:")
	assert.Contains(t, out, "name: vmess-hk-01")
	assert.Contains(t, out, "name: ss-jp-01")
}
```

- [ ] **Step 2: Run the renderer test before implementation**

Run: `go test ./tests/unit -run TestRenderMihomoInjectsNormalizedNodesIntoTemplate -v`
Expected: FAIL because the renderer package does not exist.

- [ ] **Step 3: Implement template loading and proxy injection**

```go
func MihomoTemplate(templatePath string, nodes []map[string]any) (string, error) {
	var doc map[string]any
	data, err := os.ReadFile(templatePath)
	if err != nil {
		return "", err
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return "", err
	}
	doc["proxies"] = nodes
	out, err := yaml.Marshal(doc)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
```

- [ ] **Step 4: Add `GET /subscriptions/mihomo` using the latest good snapshot from every provider**

```go
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	nodes, err := h.providers.ListLatestNodes(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out, err := render.MihomoTemplate(h.templatePath, nodes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/yaml")
	io.WriteString(w, out)
}
```

- [ ] **Step 5: Run renderer tests, refresh pipeline integration tests, and full suite**

Run: `go test ./tests/unit -run TestRenderMihomoInjectsNormalizedNodesIntoTemplate -v`
Expected: PASS

Run: `go test ./tests/integration -run TestRefreshRetainsPreviousSnapshotWhenProviderFails -v`
Expected: PASS

Run: `go test ./...`
Expected: PASS

- [ ] **Step 6: Commit Mihomo output generation**

```bash
git add internal/render/mihomo.go internal/output/http.go cmd/subhub/main.go tests/unit/render_mihomo_test.go tests/integration/refresh_pipeline_test.go
git commit -m "feat: render unified mihomo output"
```

### Task 7: Tighten Validation, Error Reporting, and Contributor-Facing Documentation

**Files:**
- Modify: `internal/provider/service.go`
- Modify: `internal/parse/subscription.go`
- Modify: `internal/provider/http.go`
- Modify: `README.md`
- Modify: `tests/integration/providers_api_test.go`
- Modify: `tests/unit/parse_subscription_test.go`

- [ ] **Step 1: Write failing tests for invalid URL input and malformed provider payloads**

```go
func TestCreateProviderRejectsInvalidURL(t *testing.T) {
	ts := newTestServer(t)
	resp := postJSON(t, ts.URL+"/providers", `{"name":"alpha","url":"not-a-url"}`)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, readBody(t, resp), "invalid provider url")
}

func TestParseSubscriptionRejectsMalformedPayload(t *testing.T) {
	_, _, err := parse.DecodeAndNormalize([]byte("%%%"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported provider payload")
}
```

- [ ] **Step 2: Run validation-focused tests to confirm they fail**

Run: `go test ./tests/integration -run TestCreateProviderRejectsInvalidURL -v`
Expected: FAIL because invalid URLs are not rejected yet.

Run: `go test ./tests/unit -run TestParseSubscriptionRejectsMalformedPayload -v`
Expected: FAIL because malformed payloads are not surfaced clearly yet.

- [ ] **Step 3: Add strict validation and explicit error messages**

```go
if _, err := url.ParseRequestURI(in.URL); err != nil {
	return Provider{}, fmt.Errorf("invalid provider url: %w", err)
}
```

```go
func parseYAML(payload []byte, format string) ([]node.Node, string, error) {
	var doc struct {
		Proxies []node.Node `yaml:"proxies"`
	}
	if err := yaml.Unmarshal(payload, &doc); err != nil {
		return nil, "", fmt.Errorf("unsupported provider payload: %w", err)
	}
	if len(doc.Proxies) == 0 {
		return nil, "", errors.New("unsupported provider payload: proxies list is empty")
	}
	return doc.Proxies, format, nil
}
```

- [ ] **Step 4: Update contributor docs with the Phase 1 API surface**

```md
## Phase 1 API

- `GET /providers`
- `POST /providers`
- `PUT /providers/{id}`
- `DELETE /providers/{id}`
- `POST /providers/{id}/refresh`
- `GET /subscriptions/mihomo`

New providers default to a 2 hour refresh interval when `refresh_interval_seconds` is omitted.
Supported upstream payload styles in Phase 1 are plain YAML and Base64-encoded YAML containing a `proxies` list.
```

- [ ] **Step 5: Run the full verification pass**

Run: `go test ./...`
Expected: PASS

Run: `go test -race ./...`
Expected: PASS

- [ ] **Step 6: Commit the hardening pass**

```bash
git add internal/provider/service.go internal/parse/subscription.go internal/provider/http.go README.md tests/integration/providers_api_test.go tests/unit/parse_subscription_test.go
git commit -m "feat: harden phase1 validation and docs"
```

## Self-Review

- Spec coverage: provider intake and repeated refresh are covered in Tasks 2, 4, and 5; local retention is covered in Task 4; normalization into one node model is covered in Task 3; unified Mihomo output is covered in Task 6; invalid input and predictable contributor behavior are covered in Task 7.
- Key constraints from the request are explicit: SQLite is the persistence core in Tasks 1 through 6, provider CRUD is the center of Task 2, the default 2-hour schedule is enforced in Tasks 1, 2, and 5, YAML/Base64 support is the entire focus of Task 3, and the normalized node contract is Mihomo-native `[]map[string]any` rather than a custom struct.
- Scope check: this stays inside Phase 1 and avoids Phase 2 features such as regex grouping, rule automation, health scoring, and UI management.
