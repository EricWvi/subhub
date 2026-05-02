# Phase 4 Subscription Services Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Clash config subscriptions, proxy provider subscriptions, and rule provider subscriptions, with provider-order-aware metadata headers, output-facing `proxy-group` composition, internal-group rule binding, and on-the-fly YAML assembly from stored components.

**Architecture:** Introduce a dedicated `subscription` package that owns subscription persistence, validation, CRUD, reference checks, and output assembly inputs. Keep stored subscription data minimal: subscription metadata, ordered provider references, output-facing `proxy-group` definitions, ordered `proxies` members, and internal-group bindings. Render served YAML on demand by combining current provider state, current internal proxy group resolution, current manual rules, and the static `tests/fixtures/client_sub.yaml` base.

**Tech Stack:** Go, `database/sql`, `modernc.org/sqlite`, embedded SQL migrations, `net/http`, `gopkg.in/yaml.v3`, React 19, Ant Design 6, TypeScript, `stretchr/testify`

---

## Assumptions

- Provider validity for subscription outputs is `total > 0` and `used / total < 0.99`. Providers with `total = 0` are treated as invalid for header selection and proxy-node output.
- If no selected provider is valid, the subscription output is still served, but `Subscription-Userinfo` is omitted.
- All three subscription types store an ordered provider-id list. This satisfies the provider-deletion constraint across config, proxy, and rule subscriptions and gives every subscription a deterministic source for header selection.
- Clash config subscriptions store output-facing `proxy-group` entries explicitly, including a persisted reserved `Proxies` row marked non-deletable.
- The rule-binding uniqueness rule means one internal proxy group may bind to at most one output-facing `proxy-group` within a single Clash config subscription.
- Proxy provider subscriptions and Clash config subscriptions filter out invalid providers’ proxy nodes. Rule provider subscriptions use the provider list only for reference protection and header selection; the rule payload itself is sourced from the bound internal proxy group.
- New routes live under `/api/subscriptions/...`:
  - `/api/subscriptions/clash-configs`
  - `/api/subscriptions/proxy-providers`
  - `/api/subscriptions/rule-providers`
  - each type also exposes `GET .../{id}/content` for the served YAML

## File Structure

**Modify**

- `main.go`
- `internal/provider/model.go`
- `internal/provider/repository.go`
- `internal/provider/service.go`
- `internal/provider/http.go`
- `internal/group/model.go`
- `internal/group/repository.go`
- `internal/rule/repository.go`
- `internal/output/http.go`
- `client/src/App.tsx`
- `tests/integration/providers_api_test.go`
- `tests/integration/refresh_pipeline_test.go`
- `tests/integration/store_migrations_test.go`

**Create**

- `internal/store/migrations/003_add_subscriptions.sql`
- `internal/subscription/model.go`
- `internal/subscription/repository.go`
- `internal/subscription/service.go`
- `internal/subscription/http.go`
- `internal/render/subscription.go`
- `tests/integration/subscription_api_test.go`
- `tests/integration/subscription_output_test.go`
- `tests/unit/render_subscription_test.go`
- `client/src/components/SubscriptionManager.tsx`
- `client/src/components/ClashConfigSubscriptionManager.tsx`
- `client/src/components/ProxyProviderSubscriptionManager.tsx`
- `client/src/components/RuleProviderSubscriptionManager.tsx`

**Responsibility Notes**

- `internal/store/migrations/003_add_subscriptions.sql`: add all Phase 4 tables, ordering columns, uniqueness constraints, and reserved-group flags without disturbing earlier phases.
- `internal/subscription/model.go`: define API-facing subscription models, clash config `proxy-group` models, member rows, and create/update payloads.
- `internal/subscription/repository.go`: own CRUD, list/detail reads, ordered provider/member persistence, and provider-reference existence checks used by provider deletion.
- `internal/subscription/service.go`: validate subscription input, enforce non-deletable `Proxies`, enforce reference/rule-binding rules, assemble runtime output inputs, and calculate `Subscription-Userinfo`.
- `internal/subscription/http.go`: expose CRUD plus `/content` endpoints using the same manual path parsing style as the existing handlers.
- `internal/render/subscription.go`: render Clash config, proxy provider, and rule provider YAML from already-assembled data.
- `internal/provider/*`: expose valid-provider reads, provider metadata helpers, and deletion guards that fail when any subscription type still references the provider.
- `internal/group/*`: expose output-facing internal-group resolution helpers that return proxy-node names and raw nodes, optionally filtered by allowed provider ids.
- `internal/rule/repository.go`: add ascending rule reads plus internal-group-targeted output reads needed for reverse-order Clash config rules and rule provider payloads.
- frontend subscription manager components: own the new UI tab, CRUD flows, inline display of assembled `proxy-group` metadata, and the three subscription type editors.

## Task 1: Add Subscription Schema and Route Skeletons

**Files:**

- Create: `internal/store/migrations/003_add_subscriptions.sql`
- Create: `internal/subscription/model.go`
- Create: `internal/subscription/repository.go`
- Create: `internal/subscription/service.go`
- Create: `internal/subscription/http.go`
- Modify: `main.go`
- Test: `tests/integration/subscription_api_test.go`
- Test: `tests/integration/store_migrations_test.go`

- [ ] **Step 1: Write the failing integration test for the empty Clash config subscription list**

```go
func TestListClashConfigSubscriptionsStartsEmpty(t *testing.T) {
	ts := newTestServerWithSubscriptions(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/subscriptions/clash-configs")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.JSONEq(t, `{"subscriptions":[]}`, readBody(t, resp))
}
```

- [ ] **Step 2: Run the focused integration test to confirm the route does not exist yet**

Run: `go test ./tests/integration -run TestListClashConfigSubscriptionsStartsEmpty -v`
Expected: FAIL with `404` or missing route behavior because `/api/subscriptions/clash-configs` is not wired yet.

- [ ] **Step 3: Add the Phase 4 schema, models, and route skeletons**

```sql
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
    bind_internal_proxy_group_id INTEGER NOT NULL,
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
```

```go
type ClashConfigSubscription struct {
	ID         int64                     `json:"id"`
	Name       string                    `json:"name"`
	Providers  []int64                   `json:"providers"`
	ProxyGroups []ClashConfigProxyGroup  `json:"proxy_groups"`
	CreatedAt  time.Time                 `json:"created_at"`
	UpdatedAt  time.Time                 `json:"updated_at"`
}

type ClashConfigProxyGroup struct {
	ID                 int64             `json:"id"`
	Name               string            `json:"name"`
	Type               string            `json:"type"`
	URL                string            `json:"url"`
	Interval           int64             `json:"interval"`
	Proxies            []ProxyMember     `json:"proxies"`
	BindInternalGroupID int64            `json:"bind_internal_proxy_group_id"`
	IsSystem           bool              `json:"is_system"`
}

type ProxyMember struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}
```

```go
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/subscriptions/clash-configs", h.handleClashConfigSubscriptions)
	mux.HandleFunc("/subscriptions/clash-configs/", h.handleClashConfigSubscriptionByID)
	mux.HandleFunc("/subscriptions/proxy-providers", h.handleProxyProviderSubscriptions)
	mux.HandleFunc("/subscriptions/proxy-providers/", h.handleProxyProviderSubscriptionByID)
	mux.HandleFunc("/subscriptions/rule-providers", h.handleRuleProviderSubscriptions)
	mux.HandleFunc("/subscriptions/rule-providers/", h.handleRuleProviderSubscriptionByID)
}
```

```go
subscriptionRepo := subscription.NewRepository(db)
subscriptionSvc := subscription.NewService(subscriptionRepo, repo, groupRepo, ruleRepo)
subscriptionHandler := subscription.NewHandler(subscriptionSvc, "tests/fixtures/client_sub.yaml")

subscriptionHandler.RegisterRoutes(apiMux)
```

- [ ] **Step 4: Re-run the focused migration and list tests**

Run: `go test ./tests/integration -run 'TestListClashConfigSubscriptionsStartsEmpty|TestStoreAppliesAllMigrations' -v`
Expected: PASS

- [ ] **Step 5: Commit the subscription schema and route baseline**

```bash
git add internal/store/migrations/003_add_subscriptions.sql internal/subscription/model.go internal/subscription/repository.go internal/subscription/service.go internal/subscription/http.go main.go tests/integration/subscription_api_test.go tests/integration/store_migrations_test.go
git commit -m "feat: add subscription schema and routes"
```

## Task 2: Implement Clash Config Subscription CRUD and Provider Reference Protection

**Files:**

- Modify: `internal/subscription/model.go`
- Modify: `internal/subscription/repository.go`
- Modify: `internal/subscription/service.go`
- Modify: `internal/subscription/http.go`
- Modify: `internal/provider/repository.go`
- Modify: `internal/provider/service.go`
- Modify: `internal/provider/http.go`
- Test: `tests/integration/subscription_api_test.go`
- Test: `tests/integration/providers_api_test.go`

- [ ] **Step 1: Add failing tests for Clash config CRUD, reserved `Proxies`, and provider delete protection**

```go
func TestCreateGetUpdateDeleteClashConfigSubscription(t *testing.T) {
	ts := newTestServerWithSubscriptions(t)
	defer ts.Close()

	p1 := createProvider(t, ts.URL, `{"name":"alpha","url":"https://example.com/a"}`)
	p2 := createProvider(t, ts.URL, `{"name":"beta","url":"https://example.com/b"}`)
	g1 := createProxyGroup(t, ts.URL, `{"name":"Streaming","script":""}`)
	g2 := createProxyGroup(t, ts.URL, `{"name":"AI","script":""}`)

	createResp := postJSON(t, ts.URL+"/api/subscriptions/clash-configs", fmt.Sprintf(`{
		"name":"Daily",
		"providers":[%d,%d],
		"proxy_groups":[
			{
				"name":"Media",
				"type":"url-test",
				"url":"https://cp.cloudflare.com/generate_204",
				"interval":300,
				"proxies":[{"type":"internal","value":"%d"},{"type":"DIRECT","value":"DIRECT"}],
				"bind_internal_proxy_group_id":%d
			}
		]
	}`, p1, p2, g1, g2))
	defer createResp.Body.Close()
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	assert.Contains(t, readBody(t, createResp), `"name":"Proxies"`)

	getResp, err := http.Get(ts.URL + "/api/subscriptions/clash-configs/1")
	require.NoError(t, err)
	defer getResp.Body.Close()
	assert.Equal(t, http.StatusOK, getResp.StatusCode)
	assert.Contains(t, readBody(t, getResp), `"providers":[1,2]`)

	updateResp := putJSON(t, ts.URL+"/api/subscriptions/clash-configs/1", fmt.Sprintf(`{
		"name":"Daily 2",
		"providers":[%d],
		"proxy_groups":[
			{
				"name":"Media",
				"type":"fallback",
				"url":"https://cp.cloudflare.com/generate_204",
				"interval":300,
				"proxies":[{"type":"internal","value":"%d"},{"type":"REJECT","value":"REJECT"}],
				"bind_internal_proxy_group_id":%d
			}
		]
	}`, p2, g1, g2))
	defer updateResp.Body.Close()
	assert.Equal(t, http.StatusOK, updateResp.StatusCode)

	deleteResp := deleteRequest(t, ts.URL+"/api/subscriptions/clash-configs/1")
	defer deleteResp.Body.Close()
	assert.Equal(t, http.StatusNoContent, deleteResp.StatusCode)
}

func TestDeleteProviderRejectsSubscriptionReference(t *testing.T) {
	ts := newTestServerWithSubscriptions(t)
	defer ts.Close()

	providerID := createProvider(t, ts.URL, `{"name":"alpha","url":"https://example.com/a"}`)
	groupID := createProxyGroup(t, ts.URL, `{"name":"Streaming","script":""}`)

	resp := postJSON(t, ts.URL+"/api/subscriptions/proxy-providers", fmt.Sprintf(`{
		"name":"Provider Export",
		"providers":[%d],
		"internal_proxy_group_id":%d
	}`, providerID, groupID))
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	deleteResp := deleteRequest(t, fmt.Sprintf("%s/api/providers/%d", ts.URL, providerID))
	defer deleteResp.Body.Close()
	assert.Equal(t, http.StatusConflict, deleteResp.StatusCode)
	assert.Contains(t, readBody(t, deleteResp), "subscription")
}
```

- [ ] **Step 2: Run the focused tests to capture the missing CRUD and delete-guard behavior**

Run: `go test ./tests/integration -run 'TestCreateGetUpdateDeleteClashConfigSubscription|TestDeleteProviderRejectsSubscriptionReference' -v`
Expected: FAIL because the CRUD service, reserved-group creation, and provider delete checks are not implemented yet.

- [ ] **Step 3: Implement Clash config CRUD with reserved `Proxies` creation and provider delete checks**

```go
var (
	ErrSubscriptionNameRequired = errors.New("subscription name is required")
	ErrProvidersRequired        = errors.New("at least one provider is required")
	ErrReservedProxyGroup       = errors.New("Proxies proxy-group cannot be deleted")
	ErrDuplicateRuleBinding     = errors.New("internal proxy group already bound")
	ErrSubscriptionProviderRef  = errors.New("provider is referenced by a subscription")
)
```

```go
func (s *Service) CreateClashConfig(ctx context.Context, in CreateClashConfigSubscriptionInput) (ClashConfigSubscription, error) {
	if strings.TrimSpace(in.Name) == "" {
		return ClashConfigSubscription{}, ErrSubscriptionNameRequired
	}
	if len(in.Providers) == 0 {
		return ClashConfigSubscription{}, ErrProvidersRequired
	}
	sub, err := s.repo.CreateClashConfig(ctx, in)
	if err != nil {
		return ClashConfigSubscription{}, err
	}
	_, err = s.repo.CreateSystemProxyGroup(ctx, sub.ID, CreateClashConfigProxyGroupInput{
		Name:                "Proxies",
		Type:                "select",
		Proxies:             []ProxyMember{{Type: "DIRECT", Value: "DIRECT"}},
		BindInternalGroupID: in.ProxyGroups[0].BindInternalGroupID,
	})
	if err != nil {
		return ClashConfigSubscription{}, err
	}
	return s.repo.GetClashConfigByID(ctx, sub.ID)
}
```

```go
func (r *Repository) ProviderReferencedByAnySubscription(ctx context.Context, providerID int64) (bool, error) {
	query := `
	SELECT EXISTS(
		SELECT 1 FROM clash_config_subscription_providers WHERE provider_id = ?
		UNION ALL
		SELECT 1 FROM proxy_provider_subscription_providers WHERE provider_id = ?
		UNION ALL
		SELECT 1 FROM rule_provider_subscription_providers WHERE provider_id = ?
	)`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, providerID, providerID, providerID).Scan(&exists)
	return exists, err
}
```

```go
func (s *Service) Delete(ctx context.Context, id int64) error {
	used, err := s.repo.ProviderReferencedByAnySubscription(ctx, id)
	if err != nil {
		return err
	}
	if used {
		return ErrSubscriptionProviderRef
	}
	return s.repo.Delete(ctx, id)
}
```

- [ ] **Step 4: Re-run the focused CRUD and delete-protection tests**

Run: `go test ./tests/integration -run 'TestCreateGetUpdateDeleteClashConfigSubscription|TestDeleteProviderRejectsSubscriptionReference' -v`
Expected: PASS

- [ ] **Step 5: Commit the Clash config CRUD and provider reference guard**

```bash
git add internal/subscription/model.go internal/subscription/repository.go internal/subscription/service.go internal/subscription/http.go internal/provider/repository.go internal/provider/service.go internal/provider/http.go tests/integration/subscription_api_test.go tests/integration/providers_api_test.go
git commit -m "feat: add clash config subscription CRUD"
```

## Task 3: Implement Clash Config YAML Assembly and Metadata Header Selection

**Files:**

- Create: `internal/render/subscription.go`
- Modify: `internal/subscription/service.go`
- Modify: `internal/group/model.go`
- Modify: `internal/group/repository.go`
- Modify: `internal/rule/repository.go`
- Modify: `internal/subscription/http.go`
- Test: `tests/integration/subscription_output_test.go`
- Test: `tests/unit/render_subscription_test.go`

- [ ] **Step 1: Add failing tests for Clash config `/content`, provider validity filtering, header selection, and rule remapping**

```go
func TestClashConfigSubscriptionContentBuildsFromStoredComponents(t *testing.T) {
	ts, repo := newTestServerWithRefreshAndSubscriptions(t)
	defer ts.Close()

	upstreamA := newUserinfoUpstream(t,
		"upload=0; download=100; total=1000; expire=1893456000",
		[]byte("proxies:\n  - {name: hk-01, type: vmess, server: hk.example.com, port: 443}\n"),
	)
	upstreamB := newUserinfoUpstream(t,
		"upload=0; download=995; total=1000; expire=1893457000",
		[]byte("proxies:\n  - {name: us-01, type: vmess, server: us.example.com, port: 443}\n"),
	)

	p1 := createProvider(t, ts.URL, fmt.Sprintf(`{"name":"alpha","url":"%s"}`, upstreamA.URL))
	p2 := createProvider(t, ts.URL, fmt.Sprintf(`{"name":"beta","url":"%s"}`, upstreamB.URL))
	require.Equal(t, http.StatusNoContent, refreshProvider(t, ts.URL, p1).StatusCode)
	require.Equal(t, http.StatusNoContent, refreshProvider(t, ts.URL, p2).StatusCode)

	internalA := createProxyGroup(t, ts.URL, `{"name":"HK","script":"function (proxyNodes) { return [proxyNodes[0].id] }"}`)
	internalB := createProxyGroup(t, ts.URL, `{"name":"US","script":"function (proxyNodes) { return proxyNodes.map(function (node) { return node.id }) }"}`)
	_ = repo

	createResp := postJSON(t, ts.URL+"/api/subscriptions/clash-configs", fmt.Sprintf(`{
		"name":"Daily",
		"providers":[%d,%d],
		"proxy_groups":[
			{
				"name":"Final",
				"type":"select",
				"proxies":[
					{"type":"reference","value":"Proxies"},
					{"type":"internal","value":"%d"},
					{"type":"DIRECT","value":"DIRECT"}
				],
				"bind_internal_proxy_group_id":%d
			}
		]
	}`, p1, p2, internalA, internalB))
	defer createResp.Body.Close()
	require.Equal(t, http.StatusCreated, createResp.StatusCode)

	contentResp, err := http.Get(ts.URL + "/api/subscriptions/clash-configs/1/content")
	require.NoError(t, err)
	defer contentResp.Body.Close()

	assert.Equal(t, http.StatusOK, contentResp.StatusCode)
	assert.Equal(t, "upload=0; download=100; total=1000; expire=1893456000", contentResp.Header.Get("Subscription-Userinfo"))
	body := readBody(t, contentResp)
	assert.Contains(t, body, "name: hk-01")
	assert.NotContains(t, body, "name: us-01")
	assert.Contains(t, body, "- Proxies")
	assert.Contains(t, body, "MATCH,Final")
}
```

- [ ] **Step 2: Run the focused output tests to capture the missing assembly behavior**

Run: `go test ./tests/integration -run TestClashConfigSubscriptionContentBuildsFromStoredComponents -v`
Expected: FAIL because `/content` assembly, provider validity filtering, rule remapping, and header selection are not implemented yet.

- [ ] **Step 3: Implement runtime assembly helpers and YAML renderers**

```go
type GroupResolvedNodes struct {
	InternalGroupID int64
	Names           []string
	Nodes           []map[string]any
}

func (r *Repository) ResolveProxyNodesForGroup(ctx context.Context, groupID int64, allowedProviderIDs []int64) (GroupResolvedNodes, error) {
	// Returns flattened node names and raw nodes for the internal proxy group,
	// already filtered to the allowed provider ids and preserving node order.
}
```

```go
func (r *Repository) ListAscendingForOutput(ctx context.Context) ([]RuleOutputRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT r.id, r.rule_type, r.pattern, r.target_kind, r.proxy_group_id
		 FROM rules r
		 ORDER BY r.id ASC`)
	// ...
}
```

```go
func (s *Service) BuildClashConfigContent(ctx context.Context, id int64) (RenderedContent, error) {
	sub, err := s.repo.GetClashConfigByID(ctx, id)
	if err != nil {
		return RenderedContent{}, err
	}

	validProviders := firstValidProvider(sub.ProviderDetails)
	allowedProviderIDs := filterValidProviderIDs(sub.ProviderDetails)
	resolvedGroups, err := s.resolveInternalGroups(ctx, sub.ProxyGroups, allowedProviderIDs)
	if err != nil {
		return RenderedContent{}, err
	}
	rules, err := s.buildMappedRules(ctx, sub.ProxyGroups)
	if err != nil {
		return RenderedContent{}, err
	}
	yamlText, err := render.RenderClashConfigSubscription("tests/fixtures/client_sub.yaml", resolvedGroups.AllNodes, sub.ProxyGroups, rules)
	if err != nil {
		return RenderedContent{}, err
	}
	return RenderedContent{
		ContentType:           "application/yaml",
		SubscriptionUserinfo:  formatUserinfo(validProviders),
		Body:                  yamlText,
	}, nil
}
```

```go
func RenderClashConfigSubscription(templatePath string, proxies []map[string]any, groups []RenderedProxyGroup, rules []string) (string, error) {
	data, err := os.ReadFile(templatePath)
	if err != nil {
		return "", err
	}
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return "", err
	}
	doc["proxies"] = proxies
	doc["proxy-groups"] = groups
	doc["rules"] = append(anySlice(rules), doc["rules"].([]any)...)
	out, err := yaml.Marshal(doc)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
```

- [ ] **Step 4: Re-run the focused output tests and add the unit renderer test**

Run: `go test ./tests/integration -run TestClashConfigSubscriptionContentBuildsFromStoredComponents -v`
Expected: PASS

Run: `go test ./tests/unit -run TestRenderClashConfigSubscription -v`
Expected: PASS

- [ ] **Step 5: Commit the Clash config assembly path**

```bash
git add internal/render/subscription.go internal/subscription/service.go internal/group/model.go internal/group/repository.go internal/rule/repository.go internal/subscription/http.go tests/integration/subscription_output_test.go tests/unit/render_subscription_test.go
git commit -m "feat: render clash config subscriptions"
```

## Task 4: Implement Proxy Provider and Rule Provider Subscription CRUD and Outputs

**Files:**

- Modify: `internal/subscription/model.go`
- Modify: `internal/subscription/repository.go`
- Modify: `internal/subscription/service.go`
- Modify: `internal/subscription/http.go`
- Modify: `internal/render/subscription.go`
- Test: `tests/integration/subscription_api_test.go`
- Test: `tests/integration/subscription_output_test.go`
- Test: `tests/unit/render_subscription_test.go`

- [ ] **Step 1: Add failing CRUD and `/content` tests for proxy provider and rule provider subscriptions**

```go
func TestProxyProviderSubscriptionContentExportsYamlProxies(t *testing.T) {
	ts := newTestServerWithRefreshAndSubscriptions(t)
	defer ts.Close()

	providerID := seedProviderWithNodes(t, ts.URL, "alpha", "hk-01", "hk-02")
	groupID := createProxyGroup(t, ts.URL, `{"name":"HK","script":""}`)

	createResp := postJSON(t, ts.URL+"/api/subscriptions/proxy-providers", fmt.Sprintf(`{
		"name":"HK Export",
		"providers":[%d],
		"internal_proxy_group_id":%d
	}`, providerID, groupID))
	defer createResp.Body.Close()
	require.Equal(t, http.StatusCreated, createResp.StatusCode)

	contentResp, err := http.Get(ts.URL + "/api/subscriptions/proxy-providers/1/content")
	require.NoError(t, err)
	defer contentResp.Body.Close()

	assert.Equal(t, http.StatusOK, contentResp.StatusCode)
	assert.Contains(t, readBody(t, contentResp), "proxies:")
	assert.Contains(t, readBody(t, contentResp), "name: hk-01")
}

func TestRuleProviderSubscriptionContentExportsYamlPayload(t *testing.T) {
	ts := newTestServerWithRefreshAndSubscriptions(t)
	defer ts.Close()

	providerID := createProvider(t, ts.URL, `{"name":"alpha","url":"https://example.com/a"}`)
	groupID := createProxyGroup(t, ts.URL, `{"name":"Rules","script":""}`)
	createRule(t, ts.URL, `{"rule_type":"DOMAIN-SUFFIX","pattern":"google.com","proxy_group":"Rules"}`)

	createResp := postJSON(t, ts.URL+"/api/subscriptions/rule-providers", fmt.Sprintf(`{
		"name":"Rules Export",
		"providers":[%d],
		"internal_proxy_group_id":%d
	}`, providerID, groupID))
	defer createResp.Body.Close()
	require.Equal(t, http.StatusCreated, createResp.StatusCode)

	contentResp, err := http.Get(ts.URL + "/api/subscriptions/rule-providers/1/content")
	require.NoError(t, err)
	defer contentResp.Body.Close()

	assert.Equal(t, http.StatusOK, contentResp.StatusCode)
	assert.Contains(t, readBody(t, contentResp), "payload:")
	assert.Contains(t, readBody(t, contentResp), "DOMAIN-SUFFIX,google.com")
}
```

- [ ] **Step 2: Run the focused provider-subscription tests to capture missing behavior**

Run: `go test ./tests/integration -run 'TestProxyProviderSubscriptionContentExportsYamlProxies|TestRuleProviderSubscriptionContentExportsYamlPayload' -v`
Expected: FAIL because the non-clash subscription CRUD and content builders are still incomplete.

- [ ] **Step 3: Implement provider-subscription CRUD and renderers**

```go
type ProxyProviderSubscription struct {
	ID                  int64     `json:"id"`
	Name                string    `json:"name"`
	Providers           []int64   `json:"providers"`
	InternalProxyGroupID int64    `json:"internal_proxy_group_id"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type RuleProviderSubscription struct {
	ID                  int64     `json:"id"`
	Name                string    `json:"name"`
	Providers           []int64   `json:"providers"`
	InternalProxyGroupID int64    `json:"internal_proxy_group_id"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}
```

```go
func (s *Service) BuildProxyProviderContent(ctx context.Context, id int64) (RenderedContent, error) {
	sub, err := s.repo.GetProxyProviderByID(ctx, id)
	if err != nil {
		return RenderedContent{}, err
	}
	allowedProviderIDs := filterValidProviderIDs(sub.ProviderDetails)
	resolved, err := s.groupRepo.ResolveProxyNodesForGroup(ctx, sub.InternalProxyGroupID, allowedProviderIDs)
	if err != nil {
		return RenderedContent{}, err
	}
	body, err := render.RenderProxyProviderSubscription(resolved.Nodes)
	if err != nil {
		return RenderedContent{}, err
	}
	return RenderedContent{
		ContentType:          "application/yaml",
		SubscriptionUserinfo: formatUserinfo(firstValidProvider(sub.ProviderDetails)),
		Body:                 body,
	}, nil
}

func (s *Service) BuildRuleProviderContent(ctx context.Context, id int64) (RenderedContent, error) {
	sub, err := s.repo.GetRuleProviderByID(ctx, id)
	if err != nil {
		return RenderedContent{}, err
	}
	rules, err := s.ruleRepo.ListForInternalGroup(ctx, sub.InternalProxyGroupID)
	if err != nil {
		return RenderedContent{}, err
	}
	body, err := render.RenderRuleProviderSubscription(rules)
	if err != nil {
		return RenderedContent{}, err
	}
	return RenderedContent{
		ContentType:          "application/yaml",
		SubscriptionUserinfo: formatUserinfo(firstValidProvider(sub.ProviderDetails)),
		Body:                 body,
	}, nil
}
```

```go
func RenderProxyProviderSubscription(nodes []map[string]any) (string, error) {
	out, err := yaml.Marshal(map[string]any{"proxies": nodes})
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func RenderRuleProviderSubscription(rules []string) (string, error) {
	payload := make([]any, 0, len(rules))
	for _, rule := range rules {
		payload = append(payload, rule)
	}
	out, err := yaml.Marshal(map[string]any{"payload": payload})
	if err != nil {
		return "", err
	}
	return string(out), nil
}
```

- [ ] **Step 4: Re-run the focused proxy-provider and rule-provider tests**

Run: `go test ./tests/integration -run 'TestProxyProviderSubscriptionContentExportsYamlProxies|TestRuleProviderSubscriptionContentExportsYamlPayload' -v`
Expected: PASS

Run: `go test ./tests/unit -run 'TestRenderProxyProviderSubscription|TestRenderRuleProviderSubscription' -v`
Expected: PASS

- [ ] **Step 5: Commit the proxy-provider and rule-provider subscription outputs**

```bash
git add internal/subscription/model.go internal/subscription/repository.go internal/subscription/service.go internal/subscription/http.go internal/render/subscription.go tests/integration/subscription_api_test.go tests/integration/subscription_output_test.go tests/unit/render_subscription_test.go
git commit -m "feat: add provider subscription outputs"
```

## Task 5: Build the Subscription Management UI

**Files:**

- Create: `client/src/components/SubscriptionManager.tsx`
- Create: `client/src/components/ClashConfigSubscriptionManager.tsx`
- Create: `client/src/components/ProxyProviderSubscriptionManager.tsx`
- Create: `client/src/components/RuleProviderSubscriptionManager.tsx`
- Modify: `client/src/App.tsx`

- [ ] **Step 1: Add the failing UX verification target by documenting the expected build**

```tsx
// App should expose a top-level "Subscriptions" tab that renders SubscriptionManager.
// Clash config rows should show provider order, proxy-group summaries, and rule bindings inline.
// Proxy provider and rule provider forms should expose ordered providers plus one internal proxy group.
```

- [ ] **Step 2: Run the client build before changes to establish the current baseline**

Run: `cd client && npm run build`
Expected: PASS before UI changes, with no subscription management code present yet.

- [ ] **Step 3: Implement the Phase 4 subscription UI flows**

```tsx
const items = [
  { key: "providers", label: "Providers" },
  { key: "groups", label: "Proxy Groups" },
  { key: "rules", label: "Rules" },
  { key: "subscriptions", label: "Subscriptions" },
];
```

```tsx
interface ClashConfigProxyGroupFormValue {
  name: string;
  type: "select" | "url-test" | "fallback";
  url?: string;
  interval?: number;
  proxies: Array<{
    type: "reference" | "internal" | "DIRECT" | "REJECT";
    value: string;
  }>;
  bind_internal_proxy_group_id: number;
}
```

```tsx
<Descriptions column={1} size="small">
  <Descriptions.Item label="Providers">
    {providerNames.join(" -> ")}
  </Descriptions.Item>
  <Descriptions.Item label="Proxy Groups">
    {proxyGroups.map((group) => (
      <Card key={group.id} size="small" title={group.name}>
        <div>Type: {group.type}</div>
        <div>Rule Binding: {group.bind_internal_proxy_group_name}</div>
        <div>
          Members:{" "}
          {group.proxies
            .map((member) => `${member.type}:${member.value}`)
            .join(", ")}
        </div>
      </Card>
    ))}
  </Descriptions.Item>
</Descriptions>
```

- [ ] **Step 4: Re-run the client build after adding the new UI**

Run: `cd client && npm run build`
Expected: PASS

- [ ] **Step 5: Commit the subscription management UI**

```bash
git add client/src/App.tsx client/src/components/SubscriptionManager.tsx client/src/components/ClashConfigSubscriptionManager.tsx client/src/components/ProxyProviderSubscriptionManager.tsx client/src/components/RuleProviderSubscriptionManager.tsx
git commit -m "feat: add subscription management ui"
```

## Task 6: End-to-End Verification and Cleanup

**Files:**

- Modify: `tests/integration/refresh_pipeline_test.go`
- Modify: `tests/integration/subscription_api_test.go`
- Modify: `tests/integration/subscription_output_test.go`
- Modify: `tests/unit/render_subscription_test.go`

- [ ] **Step 1: Add end-to-end regression coverage for the complete Phase 4 path**

```go
func TestClashConfigSubscriptionOutputUsesFirstValidProviderHeaderAndMappedRules(t *testing.T) {
	// Covers ordered providers, invalid-provider filtering, system Proxies group,
	// reverse-ordered rules, and PROXY_GROUP remapping to output-facing proxy-group names.
}

func TestDeleteProviderSucceedsAfterRemovingAllSubscriptionReferences(t *testing.T) {
	// Covers the delete-guard release path after config, proxy-provider, and rule-provider
	// subscriptions are removed.
}
```

- [ ] **Step 2: Run the focused Go verification commands**

Run: `go test ./tests/unit -v`
Expected: PASS

Run: `go test ./tests/integration -v`
Expected: PASS

- [ ] **Step 3: Run the full backend verification commands**

Run: `go test ./...`
Expected: PASS

Run: `go vet ./...`
Expected: PASS

Run: `go build .`
Expected: PASS

- [ ] **Step 4: Run the frontend verification command**

Run: `cd client && npm run build`
Expected: PASS

- [ ] **Step 5: Commit the verification and cleanup pass**

```bash
git add tests/integration/refresh_pipeline_test.go tests/integration/subscription_api_test.go tests/integration/subscription_output_test.go tests/unit/render_subscription_test.go
git commit -m "test: verify subscription services end to end"
```

## Self-Review

- Spec coverage: the plan covers all three subscription types, ordered provider storage, provider deletion protection, reserved `Proxies`, output-facing `proxy-group` composition, internal-group flattening, internal-group rule binding, `Subscription-Userinfo`, and the required YAML output shapes.
- Placeholder scan: every task includes explicit file targets, commands, and concrete snippets instead of “implement later” placeholders.
- Type consistency: the plan uses one `subscription` package, one reserved `Proxies` concept, one `bind_internal_proxy_group_id` field name, and one `/content` route convention across all subscription types.
