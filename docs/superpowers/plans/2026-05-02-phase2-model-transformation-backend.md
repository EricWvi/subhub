# Phase 2 Model Transformation Backend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Expand the backend data model and refresh pipeline so providers can store abbreviations and subscription usage metadata, while individually persisted `proxy_nodes` become the canonical source for downstream output generation.

**Architecture:** Keep the current package boundaries intact: `provider` owns schema-facing models, CRUD, and provider metadata validation; `refresh` owns fetch-to-persist orchestration; `output` reads canonical node records instead of snapshot blobs. Preserve `provider_snapshots` as last-known-good provider payload storage, but add a parallel `proxy_nodes` table plus provider usage columns so phase 2 can expose metadata without introducing phase-3 transformation logic.

**Tech Stack:** Go, `database/sql`, `modernc.org/sqlite`, embedded SQL migrations, `net/http`, `gopkg.in/yaml.v3`, `stretchr/testify`

---

## File Structure

**Modify**

- `internal/store/migrations/001_initial.sql`
- `internal/fetch/client.go`
- `internal/provider/model.go`
- `internal/provider/repository.go`
- `internal/provider/service.go`
- `internal/provider/http.go`
- `internal/refresh/service.go`
- `internal/output/http.go`
- `tests/integration/providers_api_test.go`
- `tests/integration/refresh_pipeline_test.go`

**Create**

- `tests/unit/parse_subscription_userinfo_test.go`

**Responsibility Notes**

- `internal/store/migrations/001_initial.sql`: add the phase-2 schema in place because the project currently embeds a single bootstrap migration.
- `internal/fetch/client.go`: return response headers together with the payload so refresh can parse `Subscription-Userinfo` without a second request path.
- `internal/provider/model.go`: extend provider-facing models with abbreviation and usage fields plus a node model for `proxy_nodes`.
- `internal/provider/repository.go`: own all provider metadata persistence, node mark-and-sweep persistence, and canonical node reads for output.
- `internal/provider/service.go`: validate and normalize abbreviation input for create/update flows.
- `internal/provider/http.go`: accept and return the new provider fields in existing JSON endpoints.
- `internal/refresh/service.go`: parse upstream metadata, persist snapshots, and persist nodes in one refresh transaction boundary.
- `internal/output/http.go`: switch unified output generation from snapshot-derived proxies to `proxy_nodes`.
- `tests/integration/*.go`: cover CRUD validation, refresh persistence, stale-node deletion, and output behavior.
- `tests/unit/parse_subscription_userinfo_test.go`: lock down header parsing edge cases as a pure unit test.

### Task 1: Expand the Schema and Backend Models

**Files:**

- Modify: `internal/store/migrations/001_initial.sql`
- Modify: `internal/provider/model.go`
- Test: `tests/integration/providers_api_test.go`

- [ ] **Step 1: Add a failing integration assertion for the new provider fields**

```go
func TestCreateProviderReturnsPhase2MetadataFields(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp := postJSON(t, ts.URL+"/providers", `{"name":"alpha","url":"https://example.com/sub","abbrev":"HK"}`)
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var result struct {
		Provider struct {
			Abbrev string `json:"abbrev"`
			Used   int64  `json:"used"`
			Total  int64  `json:"total"`
			Expire int64  `json:"expire"`
		} `json:"provider"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "HK", result.Provider.Abbrev)
	assert.EqualValues(t, 0, result.Provider.Used)
	assert.EqualValues(t, 0, result.Provider.Total)
	assert.EqualValues(t, 0, result.Provider.Expire)
}
```

- [ ] **Step 2: Run the test to confirm the current API does not expose phase-2 provider metadata yet**

Run: `go test ./tests/integration -run TestCreateProviderReturnsPhase2MetadataFields -v`
Expected: FAIL because `abbrev`, `used`, `total`, and `expire` are missing from the provider response and underlying schema.

- [ ] **Step 3: Extend the embedded schema with provider metadata columns and the new `proxy_nodes` table**

```sql
CREATE TABLE IF NOT EXISTS providers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    url TEXT NOT NULL,
    refresh_interval_minutes INTEGER NOT NULL,
    abbrev TEXT NOT NULL DEFAULT '',
    used INTEGER NOT NULL DEFAULT 0,
    total INTEGER NOT NULL DEFAULT 0,
    expire INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS proxy_nodes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    provider_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    raw_yaml TEXT NOT NULL,
    update_mark INTEGER NOT NULL DEFAULT 0,
    UNIQUE(provider_id, name),
    FOREIGN KEY(provider_id) REFERENCES providers(id) ON DELETE CASCADE
);
```

- [ ] **Step 4: Extend the provider and node models to match the phase-2 schema**

```go
type Provider struct {
	ID                     int64     `json:"id"`
	Name                   string    `json:"name"`
	URL                    string    `json:"url"`
	RefreshIntervalMinutes int64     `json:"refresh_interval_minutes"`
	Abbrev                 string    `json:"abbrev"`
	Used                   int64     `json:"used"`
	Total                  int64     `json:"total"`
	Expire                 int64     `json:"expire"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
	LastRefreshStatus      string    `json:"last_refresh_status,omitempty"`
	LastRefreshMessage     string    `json:"last_refresh_message,omitempty"`
}

type ProxyNode struct {
	ID         int64
	ProviderID int64
	Name       string
	RawYAML    string
	UpdateMark int64
}
```

- [ ] **Step 5: Re-run the focused integration test until the provider JSON now includes the new fields**

Run: `go test ./tests/integration -run TestCreateProviderReturnsPhase2MetadataFields -v`
Expected: PASS

- [ ] **Step 6: Commit the schema/model baseline**

```bash
git add internal/store/migrations/001_initial.sql internal/provider/model.go tests/integration/providers_api_test.go
git commit -m "feat: add phase2 provider metadata schema"
```

### Task 2: Validate and Persist Provider Abbreviations Through Existing CRUD Endpoints

**Files:**

- Modify: `internal/provider/service.go`
- Modify: `internal/provider/repository.go`
- Modify: `internal/provider/http.go`
- Test: `tests/integration/providers_api_test.go`

- [ ] **Step 1: Add failing integration tests for uppercase handling, duplicate abbreviations, and invalid input**

```go
func TestCreateProviderUppercasesAbbrevAndAllowsDuplicates(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	first := postJSON(t, ts.URL+"/providers", `{"name":"alpha","url":"https://example.com/a","abbrev":"hk"}`)
	defer first.Body.Close()
	require.Equal(t, http.StatusCreated, first.StatusCode)

	second := postJSON(t, ts.URL+"/providers", `{"name":"beta","url":"https://example.com/b","abbrev":"HK"}`)
	defer second.Body.Close()
	require.Equal(t, http.StatusCreated, second.StatusCode)

	listResp, err := http.Get(ts.URL + "/providers")
	require.NoError(t, err)
	defer listResp.Body.Close()
	assert.Contains(t, readBody(t, listResp), `"abbrev":"HK"`)
}

func TestUpdateProviderRejectsNonLetterAbbrev(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	id := createProvider(t, ts.URL, `{"name":"alpha","url":"https://example.com/sub","abbrev":"HK"}`)
	resp := putJSON(t, fmt.Sprintf("%s/providers/%d", ts.URL, id), `{"name":"alpha","url":"https://example.com/sub","abbrev":"H1"}`)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, readBody(t, resp), "uppercase letters only")
}
```

- [ ] **Step 2: Run the provider API tests to capture the missing abbreviation behavior**

Run: `go test ./tests/integration -run 'TestCreateProviderUppercasesAbbrevAndAllowsDuplicates|TestUpdateProviderRejectsNonLetterAbbrev' -v`
Expected: FAIL because the create/update payloads do not yet accept or validate `abbrev`.

- [ ] **Step 3: Extend the service inputs and validation rules for phase-2 abbreviation behavior**

```go
type CreateProviderInput struct {
	Name                   string `json:"name"`
	URL                    string `json:"url"`
	RefreshIntervalMinutes int64  `json:"refresh_interval_minutes"`
	Abbrev                 string `json:"abbrev"`
}

type UpdateProviderInput struct {
	Name                   string `json:"name"`
	URL                    string `json:"url"`
	RefreshIntervalMinutes int64  `json:"refresh_interval_minutes"`
	Abbrev                 string `json:"abbrev"`
}

var errInvalidAbbrev = errors.New("abbrev must contain uppercase letters only")

func normalizeAbbrev(raw string) (string, error) {
	upper := strings.ToUpper(strings.TrimSpace(raw))
	if upper == "" {
		return "", nil
	}
	for _, r := range upper {
		if r < 'A' || r > 'Z' {
			return "", errInvalidAbbrev
		}
	}
	return upper, nil
}
```

- [ ] **Step 4: Persist the abbreviation field in repository create, get, list, and update queries**

```go
result, err := r.db.ExecContext(ctx,
	`INSERT INTO providers
	 (name, url, refresh_interval_minutes, abbrev, used, total, expire, created_at, updated_at)
	 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
	p.Name, p.URL, p.RefreshIntervalMinutes, p.Abbrev, p.Used, p.Total, p.Expire, nowStr, nowStr,
)
```

```go
`SELECT
	p.id, p.name, p.url, p.refresh_interval_minutes, p.abbrev, p.used, p.total, p.expire,
	p.created_at, p.updated_at, ra.status, ra.message
 FROM providers p
 LEFT JOIN refresh_attempts ra ON p.id = ra.provider_id`
```

- [ ] **Step 5: Return clear `400` responses for invalid abbreviations from the HTTP layer**

```go
case errors.Is(err, errInvalidAbbrev):
	http.Error(w, err.Error(), http.StatusBadRequest)
```

- [ ] **Step 6: Re-run the provider API tests to verify the phase-2 abbreviation contract**

Run: `go test ./tests/integration -run 'TestCreateProviderUppercasesAbbrevAndAllowsDuplicates|TestUpdateProviderRejectsNonLetterAbbrev|TestCreateProviderReturnsPhase2MetadataFields' -v`
Expected: PASS

- [ ] **Step 7: Commit the provider CRUD changes**

```bash
git add internal/provider/service.go internal/provider/repository.go internal/provider/http.go tests/integration/providers_api_test.go
git commit -m "feat: add provider abbreviation support"
```

### Task 3: Parse `Subscription-Userinfo` and Persist Latest Known Usage Metadata

**Files:**

- Modify: `internal/fetch/client.go`
- Modify: `internal/provider/repository.go`
- Modify: `internal/refresh/service.go`
- Create: `tests/unit/parse_subscription_userinfo_test.go`
- Test: `tests/integration/refresh_pipeline_test.go`

- [ ] **Step 1: Add a failing unit test for header parsing and an integration test for metadata persistence**

```go
func TestParseSubscriptionUserinfo(t *testing.T) {
	meta, ok := parseSubscriptionUserinfo("upload=1; download=2; total=3; expire=1710000000")
	require.True(t, ok)
	assert.EqualValues(t, 3, meta.Used)
	assert.EqualValues(t, 3, meta.Total)
	assert.EqualValues(t, 1710000000, meta.Expire)
}
```

```go
func TestRefreshPersistsLatestSubscriptionUserinfo(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "fixtures", "provider_plain.yaml"))
	require.NoError(t, err)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Subscription-Userinfo", "upload=1024; download=2048; total=4096; expire=1893456000")
		w.Header().Set("Content-Type", "text/yaml")
		_, _ = w.Write(fixture)
	}))
	defer upstream.Close()

	ts, repo := newTestServerWithRefresh(t)
	defer ts.Close()

	providerID := createProvider(t, ts.URL, fmt.Sprintf(`{"name":"alpha","url":"%s"}`, upstream.URL))
	resp := refreshProvider(t, ts.URL, providerID)
	defer resp.Body.Close()
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	p, err := repo.GetByID(context.Background(), providerID)
	require.NoError(t, err)
	assert.EqualValues(t, 3072, p.Used)
	assert.EqualValues(t, 4096, p.Total)
	assert.EqualValues(t, 1893456000, p.Expire)
}
```

- [ ] **Step 2: Run the focused tests to prove the fetch path currently discards upstream metadata**

Run: `go test ./tests/unit -run TestParseSubscriptionUserinfo -v`
Expected: FAIL because no parser exists yet.

Run: `go test ./tests/integration -run TestRefreshPersistsLatestSubscriptionUserinfo -v`
Expected: FAIL because the fetch/refresh path does not retain response headers or provider usage metadata.

- [ ] **Step 3: Change the fetch client to return payload plus headers in a single response object**

```go
type Response struct {
	Body    []byte
	Headers http.Header
}

func (c *Client) Fetch(ctx context.Context, url string) (Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Response{}, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Response{}, fmt.Errorf("upstream returned status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, err
	}
	return Response{Body: body, Headers: resp.Header.Clone()}, nil
}
```

- [ ] **Step 4: Add a small parser for `Subscription-Userinfo` that derives `used = upload + download`**

```go
type SubscriptionInfo struct {
	Used   int64
	Total  int64
	Expire int64
}

func parseSubscriptionUserinfo(raw string) (SubscriptionInfo, bool) {
	var upload, download int64
	var meta SubscriptionInfo
	parts := strings.Split(raw, ";")
	for _, part := range parts {
		key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			return SubscriptionInfo{}, false
		}
		n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err != nil {
			return SubscriptionInfo{}, false
		}
		switch strings.TrimSpace(key) {
		case "upload":
			upload = n
		case "download":
			download = n
		case "total":
			meta.Total = n
		case "expire":
			meta.Expire = n
		}
	}
	meta.Used = upload + download
	return meta, meta.Total > 0 || meta.Expire > 0 || meta.Used > 0
}
```

- [ ] **Step 5: Persist the parsed metadata as part of the successful refresh write path**

```go
info, hasInfo := parseSubscriptionUserinfo(fetchResp.Headers.Get("Subscription-Userinfo"))

err = s.providers.ReplaceLastKnownGoodSnapshot(ctx, providerID, provider.ReplaceSnapshotInput{
	Format: format,
	Nodes:  nodes,
	Used:   info.Used,
	Total:  info.Total,
	Expire: info.Expire,
	HasUsageInfo: hasInfo,
})
```

- [ ] **Step 6: Update the repository write path so malformed or absent metadata leaves the provider row usable**

```go
if in.HasUsageInfo {
	_, err = tx.ExecContext(ctx,
		`UPDATE providers SET used = ?, total = ?, expire = ?, updated_at = ? WHERE id = ?`,
		in.Used, in.Total, in.Expire, nowStr, providerID,
	)
} else {
	_, err = tx.ExecContext(ctx,
		`UPDATE providers SET used = 0, total = 0, expire = 0, updated_at = ? WHERE id = ?`,
		nowStr, providerID,
	)
}
```

- [ ] **Step 7: Re-run the unit and integration tests for metadata parsing/persistence**

Run: `go test ./tests/unit -run TestParseSubscriptionUserinfo -v`
Expected: PASS

Run: `go test ./tests/integration -run TestRefreshPersistsLatestSubscriptionUserinfo -v`
Expected: PASS

- [ ] **Step 8: Commit the usage metadata pipeline**

```bash
git add internal/fetch/client.go internal/provider/repository.go internal/refresh/service.go tests/unit/parse_subscription_userinfo_test.go tests/integration/refresh_pipeline_test.go
git commit -m "feat: persist provider subscription metadata"
```

### Task 4: Persist `proxy_nodes` with Explicit Mark-and-Sweep Cleanup

**Files:**

- Modify: `internal/provider/repository.go`
- Modify: `internal/refresh/service.go`
- Test: `tests/integration/refresh_pipeline_test.go`

- [ ] **Step 1: Add a failing integration test that proves stale node rows are deleted only after the refresh completes**

```go
func TestRefreshReplacesProviderNodesUsingUpdateMarkSweep(t *testing.T) {
	firstPayload := []byte("proxies:\n  - {name: vmess-hk-01, type: vmess, server: hk.example.com, port: 443}\n  - {name: ss-jp-01, type: ss, server: jp.example.com, port: 443, cipher: aes-128-gcm, password: secret}\n")
	secondPayload := []byte("proxies:\n  - {name: vmess-hk-01, type: vmess, server: hk2.example.com, port: 8443}\n  - {name: trojan-sg-01, type: trojan, server: sg.example.com, port: 443, password: secret}\n")

	var current atomic.Pointer[[]byte]
	current.Store(&firstPayload)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/yaml")
		payload := current.Load()
		_, _ = w.Write(*payload)
	}))
	defer upstream.Close()

	ts, repo := newTestServerWithRefresh(t)
	defer ts.Close()

	providerID := createProvider(t, ts.URL, fmt.Sprintf(`{"name":"alpha","url":"%s"}`, upstream.URL))

	resp1 := refreshProvider(t, ts.URL, providerID)
	defer resp1.Body.Close()
	require.Equal(t, http.StatusNoContent, resp1.StatusCode)

	current.Store(&secondPayload)

	resp2 := refreshProvider(t, ts.URL, providerID)
	defer resp2.Body.Close()
	require.Equal(t, http.StatusNoContent, resp2.StatusCode)

	nodes, err := repo.ListProxyNodesByProvider(context.Background(), providerID)
	require.NoError(t, err)
	require.Len(t, nodes, 2)
	assert.Equal(t, "vmess-hk-01", nodes[0].Name)
	assert.Equal(t, int64(0), nodes[0].UpdateMark)
	assert.Contains(t, nodes[0].RawYAML, "hk2.example.com")
	assert.Equal(t, "trojan-sg-01", nodes[1].Name)
	assert.Equal(t, int64(0), nodes[1].UpdateMark)
	assert.Contains(t, nodes[1].RawYAML, "sg.example.com")
}
```

- [ ] **Step 2: Run the focused refresh test to capture the missing node-level persistence**

Run: `go test ./tests/integration -run TestRefreshReplacesProviderNodesUsingUpdateMarkSweep -v`
Expected: FAIL because `proxy_nodes` persistence and cleanup do not exist yet.

- [ ] **Step 3: Introduce a repository input type that can atomically update snapshots, usage metadata, and individual node records**

```go
type ReplaceSnapshotInput struct {
	Format       string
	Nodes        []map[string]any
	Used         int64
	Total        int64
	Expire       int64
	HasUsageInfo bool
}
```

- [ ] **Step 4: Implement the exact mark-and-sweep algorithm inside the repository transaction**

```go
_, err = tx.ExecContext(ctx, `UPDATE proxy_nodes SET update_mark = 1 WHERE provider_id = ?`, providerID)
if err != nil {
	return err
}

for _, node := range in.Nodes {
	name, _ := node["name"].(string)
	raw, err := yaml.Marshal(node)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx,
		`INSERT INTO proxy_nodes (provider_id, name, raw_yaml, update_mark)
		 VALUES (?, ?, ?, 0)
		 ON CONFLICT(provider_id, name) DO UPDATE SET
		 	raw_yaml = excluded.raw_yaml,
		 	update_mark = 0`,
		providerID, name, string(raw),
	)
	if err != nil {
		return err
	}
}

_, err = tx.ExecContext(ctx, `DELETE FROM proxy_nodes WHERE provider_id = ? AND update_mark = 1`, providerID)
if err != nil {
	return err
}
```

- [ ] **Step 5: Add a repository read helper for provider-scoped node assertions**

```go
func (r *Repository) ListProxyNodesByProvider(ctx context.Context, providerID int64) ([]ProxyNode, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, provider_id, name, raw_yaml, update_mark
		 FROM proxy_nodes
		 WHERE provider_id = ?
		 ORDER BY id`,
		providerID,
	)
```

- [ ] **Step 6: Update the refresh service to feed the new repository input without altering snapshot normalization**

```go
fetchResp, err := s.fetcher.Fetch(ctx, p.URL)
if err != nil {
	// existing failure path
}

nodes, format, err := parse.DecodeAndNormalize(fetchResp.Body)
if err != nil {
	// existing failure path
}

err = s.providers.ReplaceLastKnownGoodSnapshot(ctx, providerID, provider.ReplaceSnapshotInput{
	Format:       format,
	Nodes:        nodes,
	Used:         info.Used,
	Total:        info.Total,
	Expire:       info.Expire,
	HasUsageInfo: hasInfo,
})
```

- [ ] **Step 7: Re-run the stale-node cleanup test and a broader refresh test**

Run: `go test ./tests/integration -run 'TestRefreshReplacesProviderNodesUsingUpdateMarkSweep|TestRefreshSucceedsOnFirstFetch' -v`
Expected: PASS

- [ ] **Step 8: Commit the node persistence work**

```bash
git add internal/provider/repository.go internal/refresh/service.go tests/integration/refresh_pipeline_test.go
git commit -m "feat: persist proxy nodes individually"
```

### Task 5: Make `proxy_nodes` the Canonical Output Source and Lock In End-to-End Behavior

**Files:**

- Modify: `internal/provider/repository.go`
- Modify: `internal/output/http.go`
- Test: `tests/integration/refresh_pipeline_test.go`

- [ ] **Step 1: Add a failing integration test that proves unified output comes from current `proxy_nodes` records**

```go
func TestMihomoOutputUsesCanonicalProxyNodes(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "fixtures", "provider_plain.yaml"))
	require.NoError(t, err)

	upstream := newFlakyUpstream(t, fixture)
	ts, repo := newTestServerWithRefresh(t)
	defer ts.Close()

	providerID := createProvider(t, ts.URL, fmt.Sprintf(`{"name":"delta","url":"%s"}`, upstream.server.URL))

	refreshResp := refreshProvider(t, ts.URL, providerID)
	defer refreshResp.Body.Close()
	require.Equal(t, http.StatusNoContent, refreshResp.StatusCode)

	nodes, err := repo.ListProxyNodesByProvider(context.Background(), providerID)
	require.NoError(t, err)
	require.NotEmpty(t, nodes)

	mihomoResp, err := http.Get(ts.URL + "/subscriptions/mihomo")
	require.NoError(t, err)
	defer mihomoResp.Body.Close()

	body := readBody(t, mihomoResp)
	assert.Contains(t, body, "name: vmess-hk-01")
	assert.Contains(t, body, "name: ss-jp-01")
}
```

- [ ] **Step 2: Run the output test before swapping the canonical read path**

Run: `go test ./tests/integration -run TestMihomoOutputUsesCanonicalProxyNodes -v`
Expected: either FAIL directly or only pass incidentally through snapshot reads, demonstrating the need to switch the repository/output code intentionally.

- [ ] **Step 3: Replace snapshot-derived output reads with `proxy_nodes` deserialization**

```go
func (r *Repository) ListLatestNodes(ctx context.Context) ([]map[string]any, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT raw_yaml FROM proxy_nodes ORDER BY provider_id, id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var all []map[string]any
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var node map[string]any
		if err := yaml.Unmarshal([]byte(raw), &node); err != nil {
			return nil, err
		}
		all = append(all, node)
	}
	return all, rows.Err()
}
```

- [ ] **Step 4: Keep the output handler contract unchanged while using the new canonical repository read**

```go
nodes, err := h.providers.ListLatestNodes(r.Context())
if err != nil {
	http.Error(w, err.Error(), http.StatusInternalServerError)
	return
}
out, err := render.MihomoTemplate(h.templatePath, nodes)
```

- [ ] **Step 5: Run the focused output tests plus the full backend verification suite**

Run: `go test ./tests/integration -run 'TestMihomoOutputUsesCanonicalProxyNodes|TestMihomoOutputContainsNodesFromAllProviders|TestMihomoOutputEmptyWhenNoSnapshots' -v`
Expected: PASS

Run: `go test ./tests/unit -v`
Expected: PASS

Run: `go test ./tests/integration -v`
Expected: PASS

Run: `go vet ./...`
Expected: PASS

- [ ] **Step 6: Commit the canonical output switch**

```bash
git add internal/provider/repository.go internal/output/http.go tests/integration/refresh_pipeline_test.go tests/unit/parse_subscription_userinfo_test.go
git commit -m "feat: serve mihomo output from proxy nodes"
```

## Self-Review Checklist

- Spec coverage:
  - provider abbreviations are covered in Task 2
  - provider usage metadata is covered in Task 3
  - individual proxy node persistence plus explicit stale-node deletion is covered in Task 4
  - canonical downstream output from stored nodes is covered in Task 5
- Placeholder scan:
  - every task names exact files, concrete tests, concrete commands, and concrete code shapes
- Type consistency:
  - plan uses `refresh_interval_minutes` consistently
  - plan uses `used`, `total`, and `expire` as provider fields
  - plan uses `raw_yaml` and `update_mark` consistently for `proxy_nodes`
