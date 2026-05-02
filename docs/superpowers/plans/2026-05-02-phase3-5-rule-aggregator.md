# Phase 3.5 Rule Aggregator Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add manual rule CRUD, paginated descending rule-list APIs, proxy-group-aware target validation, proxy-group deletion cascade behavior, Mihomo rule injection, and a web UI for browsing and editing rules.

**Architecture:** Introduce a dedicated `rule` package that owns manual rule persistence, validation, pagination, and HTTP CRUD. Keep the user-facing rule model as `rule_type + pattern + proxy_group`, but store user-defined proxy-group targets by stable group id internally so proxy-group deletion can cascade safely and group renames never orphan rules. Extend the existing Mihomo render path to prepend manual rules ahead of the template’s existing rules, and add a `RuleManager` client component that uses server-side pagination and always presents rules newest-first.

**Tech Stack:** Go, `database/sql`, `modernc.org/sqlite`, embedded SQL migrations, `net/http`, `gopkg.in/yaml.v3`, React 19, Ant Design 6, TypeScript, `stretchr/testify`

---

## Assumptions

- Descending rule order means newest-first using `id DESC`.
- `GET /api/rules` uses 1-based pagination with query parameters `page` and `page_size`.
- Default pagination is `page=1` and `page_size=20`.
- Maximum `page_size` is `100`.
- Manual rules are rendered before template rules in the generated Mihomo output so manual rules take precedence.
- `rule_type` validation in this phase is limited to non-empty input plus the web-page affordance for `DOMAIN-SUFFIX`, `DOMAIN-KEYWORD`, and custom text.
- The client has no automated test harness yet, so frontend verification in this plan uses `npm run build`.

## File Structure

**Modify**

- `main.go`
- `internal/store/migrations/001_initial.sql`
- `internal/group/repository.go`
- `internal/group/service.go`
- `internal/group/http.go`
- `internal/output/http.go`
- `internal/render/mihomo.go`
- `tests/integration/group_api_test.go`
- `tests/integration/refresh_pipeline_test.go`
- `tests/unit/render_mihomo_test.go`
- `client/src/App.tsx`

**Create**

- `internal/rule/model.go`
- `internal/rule/repository.go`
- `internal/rule/service.go`
- `internal/rule/http.go`
- `tests/integration/rule_api_test.go`
- `client/src/components/RuleManager.tsx`

**Responsibility Notes**

- `internal/store/migrations/001_initial.sql`: add the manual-rule schema and the internal target representation needed for safe proxy-group cascade behavior.
- `internal/rule/model.go`: define API-facing rule models, pagination metadata, and create/update inputs.
- `internal/rule/repository.go`: own rule CRUD, descending paginated reads, proxy-group target resolution, and output-facing rule string generation.
- `internal/rule/service.go`: validate non-empty fields, normalize pagination defaults, and map user-facing `proxy_group` values to internal storage.
- `internal/rule/http.go`: expose `/rules` CRUD using the same manual path parsing style as `provider/http.go` and `group/http.go`.
- `internal/group/repository.go` and `internal/group/service.go`: expose lookup helpers needed by rule validation and preserve safe behavior when groups are deleted.
- `internal/output/http.go` and `internal/render/mihomo.go`: merge manual rules into the generated Mihomo config without changing provider-node injection.
- `tests/integration/rule_api_test.go`: cover rule CRUD, pagination, descending order, proxy-group validation, and delete cascade behavior.
- `tests/integration/group_api_test.go`: cover the cross-package deletion contract from the proxy-group side.
- `tests/unit/render_mihomo_test.go`: lock down manual rule injection order without requiring a full HTTP path.
- `client/src/components/RuleManager.tsx`: own paginated rule browsing plus create/edit/delete flows.
- `client/src/App.tsx`: add a Rules tab alongside Providers and Proxy Groups.

## Task 1: Add the Rule Schema and Route Skeleton

**Files:**

- Modify: `internal/store/migrations/001_initial.sql`
- Modify: `main.go`
- Create: `internal/rule/model.go`
- Create: `internal/rule/repository.go`
- Create: `internal/rule/service.go`
- Create: `internal/rule/http.go`
- Test: `tests/integration/rule_api_test.go`

- [ ] **Step 1: Write the failing integration test for the empty paginated rule list**

```go
func TestListRulesStartsEmpty(t *testing.T) {
	ts := newTestServerWithRules(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/rules")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.JSONEq(t, `{"rules":[],"page":1,"page_size":20,"total":0}`, readBody(t, resp))
}
```

- [ ] **Step 2: Run the focused integration test to confirm the route does not exist yet**

Run: `go test ./tests/integration -run TestListRulesStartsEmpty -v`
Expected: FAIL with `404` or missing route behavior because `/api/rules` is not wired yet.

- [ ] **Step 3: Add the schema and route skeleton for manual rules**

```sql
CREATE TABLE IF NOT EXISTS rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    rule_type TEXT NOT NULL,
    pattern TEXT NOT NULL,
    target_kind TEXT NOT NULL,
    proxy_group_id INTEGER,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY(proxy_group_id) REFERENCES proxy_groups(id) ON DELETE CASCADE
);
```

```go
type Rule struct {
	ID         int64     `json:"id"`
	RuleType   string    `json:"rule_type"`
	Pattern    string    `json:"pattern"`
	ProxyGroup string    `json:"proxy_group"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type ListRulesResult struct {
	Rules    []Rule `json:"rules"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	Total    int    `json:"total"`
}
```

```go
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/rules", h.handleRules)
	mux.HandleFunc("/rules/", h.handleRuleByID)
}
```

```go
ruleRepo := rule.NewRepository(db)
ruleSvc := rule.NewService(ruleRepo)
ruleHandler := rule.NewHandler(ruleSvc)
ruleHandler.RegisterRoutes(apiMux)
```

- [ ] **Step 4: Re-run the focused test until the empty rule list route passes**

Run: `go test ./tests/integration -run TestListRulesStartsEmpty -v`
Expected: PASS

- [ ] **Step 5: Commit the schema and route baseline**

```bash
git add internal/store/migrations/001_initial.sql internal/rule/model.go internal/rule/repository.go internal/rule/service.go internal/rule/http.go main.go tests/integration/rule_api_test.go
git commit -m "feat: add rule schema and routes"
```

## Task 2: Implement Rule CRUD and Proxy-Group Target Validation

**Files:**

- Modify: `internal/rule/model.go`
- Modify: `internal/rule/repository.go`
- Modify: `internal/rule/service.go`
- Modify: `internal/rule/http.go`
- Test: `tests/integration/rule_api_test.go`

- [ ] **Step 1: Add failing CRUD and validation tests**

```go
func TestCreateGetUpdateDeleteRule(t *testing.T) {
	ts := newTestServerWithRules(t)
	defer ts.Close()

	groupID := createProxyGroup(t, ts.URL, `{"name":"Streaming","script":""}`)

	createResp := postJSON(t, ts.URL+"/api/rules", `{"rule_type":"DOMAIN-SUFFIX","pattern":"netflix.com","proxy_group":"Streaming"}`)
	defer createResp.Body.Close()
	require.Equal(t, http.StatusCreated, createResp.StatusCode)

	var created struct {
		Rule struct {
			ID         int64  `json:"id"`
			RuleType   string `json:"rule_type"`
			Pattern    string `json:"pattern"`
			ProxyGroup string `json:"proxy_group"`
		} `json:"rule"`
	}
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&created))
	assert.Equal(t, "DOMAIN-SUFFIX", created.Rule.RuleType)
	assert.Equal(t, "netflix.com", created.Rule.Pattern)
	assert.Equal(t, "Streaming", created.Rule.ProxyGroup)

	getResp, err := http.Get(fmt.Sprintf("%s/api/rules/%d", ts.URL, created.Rule.ID))
	require.NoError(t, err)
	defer getResp.Body.Close()
	assert.Equal(t, http.StatusOK, getResp.StatusCode)

	updateResp := putJSON(t, fmt.Sprintf("%s/api/rules/%d", ts.URL, created.Rule.ID), `{"rule_type":"DOMAIN-KEYWORD","pattern":"openai","proxy_group":"DIRECT"}`)
	defer updateResp.Body.Close()
	assert.Equal(t, http.StatusOK, updateResp.StatusCode)
	assert.Contains(t, readBody(t, updateResp), `"proxy_group":"DIRECT"`)

	deleteResp := deleteRequest(t, fmt.Sprintf("%s/api/rules/%d", ts.URL, created.Rule.ID))
	defer deleteResp.Body.Close()
	assert.Equal(t, http.StatusNoContent, deleteResp.StatusCode)

	_ = groupID
}

func TestCreateRuleRejectsUnknownProxyGroup(t *testing.T) {
	ts := newTestServerWithRules(t)
	defer ts.Close()

	resp := postJSON(t, ts.URL+"/api/rules", `{"rule_type":"DOMAIN-SUFFIX","pattern":"google.com","proxy_group":"Missing"}`)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, readBody(t, resp), "invalid proxy group")
}
```

- [ ] **Step 2: Run the focused tests to capture the missing CRUD and validation behavior**

Run: `go test ./tests/integration -run 'TestCreateGetUpdateDeleteRule|TestCreateRuleRejectsUnknownProxyGroup' -v`
Expected: FAIL because the rule repository and service do not yet persist rules or validate proxy-group targets.

- [ ] **Step 3: Add the rule inputs, validation errors, and target-resolution logic**

```go
type CreateRuleInput struct {
	RuleType   string `json:"rule_type"`
	Pattern    string `json:"pattern"`
	ProxyGroup string `json:"proxy_group"`
}

type UpdateRuleInput struct {
	RuleType   string `json:"rule_type"`
	Pattern    string `json:"pattern"`
	ProxyGroup string `json:"proxy_group"`
}

var (
	ErrRuleTypeRequired   = errors.New("rule type is required")
	ErrPatternRequired    = errors.New("pattern is required")
	ErrProxyGroupRequired = errors.New("proxy group is required")
	ErrInvalidProxyGroup  = errors.New("invalid proxy group")
)
```

```go
func (s *Service) resolveTarget(ctx context.Context, proxyGroup string) (string, sql.NullInt64, error) {
	switch strings.TrimSpace(proxyGroup) {
	case "":
		return "", sql.NullInt64{}, ErrProxyGroupRequired
	case "DIRECT", "REJECT":
		return strings.TrimSpace(proxyGroup), sql.NullInt64{}, nil
	default:
		id, err := s.repo.FindProxyGroupIDByName(ctx, strings.TrimSpace(proxyGroup))
		if err != nil {
			return "", sql.NullInt64{}, ErrInvalidProxyGroup
		}
		return "PROXY_GROUP", sql.NullInt64{Int64: id, Valid: true}, nil
	}
}
```

- [ ] **Step 4: Implement repository CRUD using internal target storage but API-facing `proxy_group` strings**

```go
func (r *Repository) Create(ctx context.Context, in CreateRuleRecord) (Rule, error) {
	now := nowInLocation()
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO rules (rule_type, pattern, target_kind, proxy_group_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		in.RuleType, in.Pattern, in.TargetKind, in.ProxyGroupID, now.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	if err != nil {
		return Rule{}, err
	}
	id, _ := result.LastInsertId()
	return r.GetByID(ctx, id)
}
```

```go
func (r *Repository) GetByID(ctx context.Context, id int64) (Rule, error) {
	var rule Rule
	var targetKind string
	var proxyGroupID sql.NullInt64
	var groupName sql.NullString
	var createdAt, updatedAt string

	err := r.db.QueryRowContext(ctx,
		`SELECT r.id, r.rule_type, r.pattern, r.target_kind, r.proxy_group_id, pg.name, r.created_at, r.updated_at
		 FROM rules r
		 LEFT JOIN proxy_groups pg ON pg.id = r.proxy_group_id
		 WHERE r.id = ?`,
		id,
	).Scan(&rule.ID, &rule.RuleType, &rule.Pattern, &targetKind, &proxyGroupID, &groupName, &createdAt, &updatedAt)
	if err != nil {
		return Rule{}, err
	}
	if targetKind == "PROXY_GROUP" {
		rule.ProxyGroup = groupName.String
	} else {
		rule.ProxyGroup = targetKind
	}
	rule.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	rule.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return rule, nil
}
```

- [ ] **Step 5: Re-run the focused tests until CRUD and validation pass**

Run: `go test ./tests/integration -run 'TestCreateGetUpdateDeleteRule|TestCreateRuleRejectsUnknownProxyGroup' -v`
Expected: PASS

- [ ] **Step 6: Commit the CRUD and validation work**

```bash
git add internal/rule/model.go internal/rule/repository.go internal/rule/service.go internal/rule/http.go tests/integration/rule_api_test.go
git commit -m "feat: add manual rule CRUD"
```

## Task 3: Add Descending Pagination and Proxy-Group Delete Cascade Coverage

**Files:**

- Modify: `internal/rule/repository.go`
- Modify: `internal/rule/service.go`
- Modify: `internal/rule/http.go`
- Modify: `tests/integration/rule_api_test.go`
- Modify: `tests/integration/group_api_test.go`

- [ ] **Step 1: Add failing pagination and cascade tests**

```go
func TestListRulesUsesDescendingPagination(t *testing.T) {
	ts := newTestServerWithRules(t)
	defer ts.Close()

	groupID := createProxyGroup(t, ts.URL, `{"name":"OpenAI","script":""}`)
	_ = groupID

	postJSON(t, ts.URL+"/api/rules", `{"rule_type":"DOMAIN-SUFFIX","pattern":"one.com","proxy_group":"DIRECT"}`).Body.Close()
	postJSON(t, ts.URL+"/api/rules", `{"rule_type":"DOMAIN-SUFFIX","pattern":"two.com","proxy_group":"DIRECT"}`).Body.Close()
	postJSON(t, ts.URL+"/api/rules", `{"rule_type":"DOMAIN-SUFFIX","pattern":"three.com","proxy_group":"DIRECT"}`).Body.Close()

	resp, err := http.Get(ts.URL + "/api/rules?page=1&page_size=2")
	require.NoError(t, err)
	defer resp.Body.Close()

	var result struct {
		Rules []struct {
			Pattern string `json:"pattern"`
		} `json:"rules"`
		Page     int `json:"page"`
		PageSize int `json:"page_size"`
		Total    int `json:"total"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	require.Len(t, result.Rules, 2)
	assert.Equal(t, "three.com", result.Rules[0].Pattern)
	assert.Equal(t, "two.com", result.Rules[1].Pattern)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 2, result.PageSize)
	assert.Equal(t, 3, result.Total)
}

func TestDeleteProxyGroupDeletesRelatedRules(t *testing.T) {
	ts := newTestServerWithRules(t)
	defer ts.Close()

	groupID := createProxyGroup(t, ts.URL, `{"name":"Streaming","script":""}`)
	postJSON(t, ts.URL+"/api/rules", `{"rule_type":"DOMAIN-SUFFIX","pattern":"netflix.com","proxy_group":"Streaming"}`).Body.Close()

	deleteResp := deleteRequest(t, fmt.Sprintf("%s/api/proxy-groups/%d", ts.URL, groupID))
	defer deleteResp.Body.Close()
	require.Equal(t, http.StatusNoContent, deleteResp.StatusCode)

	listResp, err := http.Get(ts.URL + "/api/rules")
	require.NoError(t, err)
	defer listResp.Body.Close()
	assert.JSONEq(t, `{"rules":[],"page":1,"page_size":20,"total":0}`, readBody(t, listResp))
}
```

- [ ] **Step 2: Run the focused tests to capture missing order, pagination, and cascade behavior**

Run: `go test ./tests/integration -run 'TestListRulesUsesDescendingPagination|TestDeleteProxyGroupDeletesRelatedRules' -v`
Expected: FAIL because list pagination metadata and proxy-group-backed cascade behavior are not implemented yet.

- [ ] **Step 3: Implement descending paginated reads and input normalization**

```go
type ListRulesInput struct {
	Page     int
	PageSize int
}

func normalizeListInput(in ListRulesInput) ListRulesInput {
	if in.Page < 1 {
		in.Page = 1
	}
	if in.PageSize < 1 {
		in.PageSize = 20
	}
	if in.PageSize > 100 {
		in.PageSize = 100
	}
	return in
}
```

```go
func (r *Repository) List(ctx context.Context, in ListRulesInput) (ListRulesResult, error) {
	in = normalizeListInput(in)
	offset := (in.Page - 1) * in.PageSize

	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM rules`).Scan(&total); err != nil {
		return ListRulesResult{}, err
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT r.id, r.rule_type, r.pattern, r.target_kind, pg.name, r.created_at, r.updated_at
		 FROM rules r
		 LEFT JOIN proxy_groups pg ON pg.id = r.proxy_group_id
		 ORDER BY r.id DESC
		 LIMIT ? OFFSET ?`,
		in.PageSize, offset,
	)
	if err != nil {
		return ListRulesResult{}, err
	}
	defer rows.Close()

	return scanRulePage(rows, in.Page, in.PageSize, total)
}
```

- [ ] **Step 4: Parse pagination query parameters in the rule handler**

```go
func parseListRulesInput(r *http.Request) ListRulesInput {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	return ListRulesInput{Page: page, PageSize: pageSize}
}
```

```go
func (h *Handler) listRules(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.List(r.Context(), parseListRulesInput(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
```

- [ ] **Step 5: Re-run the focused tests until pagination and cascade behavior pass**

Run: `go test ./tests/integration -run 'TestListRulesUsesDescendingPagination|TestDeleteProxyGroupDeletesRelatedRules' -v`
Expected: PASS

- [ ] **Step 6: Commit the pagination and cascade work**

```bash
git add internal/rule/repository.go internal/rule/service.go internal/rule/http.go tests/integration/rule_api_test.go tests/integration/group_api_test.go
git commit -m "feat: add paginated descending rule listing"
```

## Task 4: Inject Manual Rules Into Mihomo Output

**Files:**

- Modify: `internal/rule/repository.go`
- Modify: `internal/output/http.go`
- Modify: `internal/render/mihomo.go`
- Modify: `tests/integration/refresh_pipeline_test.go`
- Modify: `tests/unit/render_mihomo_test.go`

- [ ] **Step 1: Add failing render and integration tests for manual rule injection**

```go
func TestRenderMihomoPrependsManualRulesBeforeTemplateRules(t *testing.T) {
	nodes := []map[string]any{
		{"name": "test-node", "type": "vmess", "server": "test.example.com", "port": 443},
	}
	rules := []string{
		"DOMAIN-SUFFIX,openai.com,DIRECT",
		"DOMAIN-KEYWORD,netflix,REJECT",
	}

	out, err := render.MihomoTemplate("../../tests/fixtures/template.yaml", nodes, rules)
	require.NoError(t, err)

	firstManual := strings.Index(out, "DOMAIN-SUFFIX,openai.com,DIRECT")
	firstTemplate := strings.Index(out, "DOMAIN-SUFFIX,google.com,PROXY")
	assert.NotEqual(t, -1, firstManual)
	assert.NotEqual(t, -1, firstTemplate)
	assert.Less(t, firstManual, firstTemplate)
}
```

```go
func TestMihomoOutputIncludesManualRules(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "fixtures", "provider_plain.yaml"))
	require.NoError(t, err)

	upstream := newFlakyUpstream(t, fixture)
	ts, _ := newTestServerWithRefreshAndRules(t)
	defer ts.Close()

	providerID := createProvider(t, ts.URL, fmt.Sprintf(`{"name":"alpha","url":"%s"}`, upstream.server.URL))
	refreshResp := refreshProvider(t, ts.URL, providerID)
	defer refreshResp.Body.Close()
	require.Equal(t, http.StatusNoContent, refreshResp.StatusCode)

	postJSON(t, ts.URL+"/api/rules", `{"rule_type":"DOMAIN-SUFFIX","pattern":"openai.com","proxy_group":"DIRECT"}`).Body.Close()

	resp, err := http.Get(ts.URL + "/api/subscriptions/mihomo")
	require.NoError(t, err)
	defer resp.Body.Close()

	body := readBody(t, resp)
	assert.Contains(t, body, "DOMAIN-SUFFIX,openai.com,DIRECT")
	assert.Contains(t, body, "DOMAIN-SUFFIX,google.com,PROXY")
}
```

- [ ] **Step 2: Run the focused tests to capture the missing output behavior**

Run: `go test ./tests/unit -run TestRenderMihomoPrependsManualRulesBeforeTemplateRules -v && go test ./tests/integration -run TestMihomoOutputIncludesManualRules -v`
Expected: FAIL because the render path only injects proxies and ignores manual rules.

- [ ] **Step 3: Add an output-facing repository method and pass rules through the output handler**

```go
func (r *Repository) ListForOutput(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT r.rule_type, r.pattern,
		        CASE WHEN r.target_kind = 'PROXY_GROUP' THEN pg.name ELSE r.target_kind END AS proxy_group
		 FROM rules r
		 LEFT JOIN proxy_groups pg ON pg.id = r.proxy_group_id
		 ORDER BY r.id DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []string
	for rows.Next() {
		var ruleType, pattern, proxyGroup string
		if err := rows.Scan(&ruleType, &pattern, &proxyGroup); err != nil {
			return nil, err
		}
		rules = append(rules, ruleType+","+pattern+","+proxyGroup)
	}
	return rules, rows.Err()
}
```

```go
type Handler struct {
	providers    *provider.Repository
	rules        *rule.Repository
	templatePath string
}

func NewHandler(providers *provider.Repository, rules *rule.Repository, templatePath string) *Handler {
	return &Handler{providers: providers, rules: rules, templatePath: templatePath}
}
```

- [ ] **Step 4: Update the renderer to prepend manual rules before template rules**

```go
func MihomoTemplate(templatePath string, nodes []map[string]any, manualRules []string) (string, error) {
	data, err := os.ReadFile(templatePath)
	if err != nil {
		return "", err
	}

	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return "", err
	}

	existing, _ := doc["rules"].([]any)
	merged := make([]any, 0, len(manualRules)+len(existing))
	for _, rule := range manualRules {
		merged = append(merged, rule)
	}
	merged = append(merged, existing...)
	doc["rules"] = merged
	doc["proxies"] = nodes

	out, err := yaml.Marshal(doc)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
```

- [ ] **Step 5: Re-run the focused tests until output injection passes**

Run: `go test ./tests/unit -run TestRenderMihomoPrependsManualRulesBeforeTemplateRules -v && go test ./tests/integration -run TestMihomoOutputIncludesManualRules -v`
Expected: PASS

- [ ] **Step 6: Commit the rule injection work**

```bash
git add internal/rule/repository.go internal/output/http.go internal/render/mihomo.go tests/integration/refresh_pipeline_test.go tests/unit/render_mihomo_test.go main.go
git commit -m "feat: inject manual rules into mihomo output"
```

## Task 5: Build the Rule Management Web UI

**Files:**

- Create: `client/src/components/RuleManager.tsx`
- Modify: `client/src/App.tsx`

- [ ] **Step 1: Add the new client component skeleton and app tab wiring**

```tsx
import React, { useEffect, useState } from 'react';
import { Table, Button, Space, Modal, Form, Input, Select, message, Popconfirm, Typography } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';

const { Title } = Typography;

interface Rule {
  id: number;
  rule_type: string;
  pattern: string;
  proxy_group: string;
  created_at: string;
  updated_at: string;
}

const BUILT_IN_RULE_TYPES = ['DOMAIN-SUFFIX', 'DOMAIN-KEYWORD'];
const STATIC_PROXY_GROUPS = ['DIRECT', 'REJECT'];
```

```tsx
items={[
  { key: 'providers', label: 'Providers' },
  { key: 'groups', label: 'Proxy Groups' },
  { key: 'rules', label: 'Rules' },
]}
```

```tsx
{currentTab === 'providers' ? <ProviderManager /> : currentTab === 'groups' ? <ProxyGroupManager /> : <RuleManager />}
```

- [ ] **Step 2: Implement paginated rule fetching in descending order**

```tsx
const [rules, setRules] = useState<Rule[]>([]);
const [loading, setLoading] = useState(false);
const [page, setPage] = useState(1);
const [pageSize, setPageSize] = useState(20);
const [total, setTotal] = useState(0);

const fetchRules = async (nextPage = page, nextPageSize = pageSize) => {
  setLoading(true);
  try {
    const response = await fetch(`/api/rules?page=${nextPage}&page_size=${nextPageSize}`);
    const data = await response.json();
    setRules(data.rules || []);
    setPage(data.page || nextPage);
    setPageSize(data.page_size || nextPageSize);
    setTotal(data.total || 0);
  } catch (error) {
    message.error('Failed to fetch rules');
  } finally {
    setLoading(false);
  }
};

useEffect(() => {
  fetchRules(1, pageSize);
}, []);
```

- [ ] **Step 3: Implement the proxy-group option loading and rule-type custom input flow**

```tsx
const [proxyGroupOptions, setProxyGroupOptions] = useState<string[]>(STATIC_PROXY_GROUPS);
const [customRuleType, setCustomRuleType] = useState(false);

const fetchProxyGroupOptions = async () => {
  const response = await fetch('/api/proxy-groups');
  const data = await response.json();
  const dynamicNames = (data.groups || []).map((group: { name: string }) => group.name);
  setProxyGroupOptions([...STATIC_PROXY_GROUPS, ...dynamicNames]);
};
```

```tsx
<Form.Item name="rule_type_selector" label="Rule Type" rules={[{ required: true, message: 'Please select rule type' }]}>
  <Select
    options={[
      { label: 'DOMAIN-SUFFIX', value: 'DOMAIN-SUFFIX' },
      { label: 'DOMAIN-KEYWORD', value: 'DOMAIN-KEYWORD' },
      { label: 'Custom', value: '__custom__' },
    ]}
    onChange={(value) => {
      const useCustom = value === '__custom__';
      setCustomRuleType(useCustom);
      form.setFieldValue('rule_type', useCustom ? '' : value);
    }}
  />
</Form.Item>

<Form.Item name="rule_type" label="Custom Rule Type" hidden={!customRuleType} rules={[{ required: customRuleType, message: 'Please input custom rule type' }]}>
  <Input placeholder="GEOIP" />
</Form.Item>
```

- [ ] **Step 4: Implement create, edit, delete, and paginated table behavior**

```tsx
<Table
  rowKey="id"
  loading={loading}
  dataSource={rules}
  columns={[
    { title: 'Rule Type', dataIndex: 'rule_type', key: 'rule_type' },
    { title: 'Pattern', dataIndex: 'pattern', key: 'pattern' },
    { title: 'Proxy Group', dataIndex: 'proxy_group', key: 'proxy_group' },
    { title: 'Updated At', dataIndex: 'updated_at', key: 'updated_at', render: formatDate24h },
    {
      title: 'Action',
      key: 'action',
      render: (_value, record) => (
        <Space>
          <Button icon={<EditOutlined />} onClick={() => handleEdit(record)} />
          <Popconfirm title="Sure to delete?" onConfirm={() => handleDelete(record.id)}>
            <Button danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ]}
  pagination={{
    current: page,
    pageSize,
    total,
    onChange: (nextPage, nextPageSize) => fetchRules(nextPage, nextPageSize),
  }}
/>
```

```tsx
const handleDelete = async (id: number) => {
  const response = await fetch(`/api/rules/${id}`, { method: 'DELETE' });
  if (!response.ok) {
    message.error(`Delete failed: ${await response.text()}`);
    return;
  }
  message.success('Rule deleted');
  fetchRules(page, pageSize);
};
```

- [ ] **Step 5: Build the client to catch TypeScript and JSX mistakes**

Run: `cd client && npm run build`
Expected: PASS with a Vite production build emitted under `client/dist`.

- [ ] **Step 6: Commit the UI work**

```bash
git add client/src/App.tsx client/src/components/RuleManager.tsx
git commit -m "feat: add rule management ui"
```

## Task 6: Run Full Verification and Clean Up Mismatched Helpers

**Files:**

- Modify: `tests/integration/rule_api_test.go`
- Modify: `tests/integration/group_api_test.go`
- Modify: `tests/integration/refresh_pipeline_test.go`
- Modify: `tests/unit/render_mihomo_test.go`

- [ ] **Step 1: Ensure the integration helper wiring covers providers, groups, rules, and output in one test server**

```go
func newTestServerWithRefreshAndRules(t *testing.T) (*httptest.Server, *provider.Repository) {
	t.Helper()
	dir := t.TempDir()
	cfg := config.Config{
		ListenAddr:             ":0",
		DatabasePath:           filepath.Join(dir, "test.db"),
		UpstreamRequestTimeout: 5 * time.Second,
		DefaultRefreshInterval: config.DefaultRefreshInterval,
	}
	db := store.MustOpen(cfg.DatabasePath)
	t.Cleanup(func() { db.Close() })

	providerRepo := provider.NewRepository(db)
	providerSvc := provider.NewService(providerRepo)
	providerHandler := provider.NewHandler(providerSvc)

	ruleRepo := rule.NewRepository(db)
	ruleSvc := rule.NewService(ruleRepo)
	ruleHandler := rule.NewHandler(ruleSvc)

	groupRepo := group.NewRepository(db)
	groupSvc := group.NewService(groupRepo)
	groupHandler := group.NewHandler(groupSvc)

	fetcher := fetch.NewClient(cfg.UpstreamRequestTimeout)
	refreshSvc := refresh.NewService(providerRepo, fetcher)
	providerHandler.SetRefresher(refreshSvc.RefreshProvider)

	outputHandler := output.NewHandler(providerRepo, ruleRepo, filepath.Join("..", "fixtures", "template.yaml"))

	apiMux := http.NewServeMux()
	providerHandler.RegisterRoutes(apiMux)
	groupHandler.RegisterRoutes(apiMux)
	ruleHandler.RegisterRoutes(apiMux)
	outputHandler.RegisterRoutes(apiMux)

	mux := http.NewServeMux()
	mux.Handle("/api/", http.StripPrefix("/api", apiMux))
	return httptest.NewServer(mux), providerRepo
}
```

- [ ] **Step 2: Run the rule-specific backend tests**

Run: `go test ./tests/integration -run 'TestListRulesStartsEmpty|TestCreateGetUpdateDeleteRule|TestCreateRuleRejectsUnknownProxyGroup|TestListRulesUsesDescendingPagination|TestDeleteProxyGroupDeletesRelatedRules|TestMihomoOutputIncludesManualRules' -v`
Expected: PASS

- [ ] **Step 3: Run the focused unit render tests**

Run: `go test ./tests/unit -run 'TestRenderMihomoInjectsNormalizedNodesIntoTemplate|TestRenderMihomoPreservesProxyGroupsAndRules|TestRenderMihomoPrependsManualRulesBeforeTemplateRules' -v`
Expected: PASS

- [ ] **Step 4: Run the project-wide verification commands**

Run: `go test ./...`
Expected: PASS

Run: `go vet ./...`
Expected: PASS

Run: `cd client && npm run build`
Expected: PASS

- [ ] **Step 5: Commit any final helper or assertion cleanup**

```bash
git add tests/integration/rule_api_test.go tests/integration/group_api_test.go tests/integration/refresh_pipeline_test.go tests/unit/render_mihomo_test.go main.go internal/output/http.go internal/render/mihomo.go
git commit -m "test: verify phase3.5 rule aggregator"
```

## Spec Coverage Check

- Manual rule CRUD is implemented in Task 2 and verified again in Task 6.
- Rule structure as `rule_type + pattern + proxy_group` is implemented in Task 2 and preserved through Task 5.
- Finite web-page `proxy_group` choices are implemented in Task 5 by combining `DIRECT`, `REJECT`, and `/api/proxy-groups` names.
- Built-in `rule_type` choices plus custom input are implemented in Task 5.
- Proxy-group deletion cascade is implemented in Tasks 2 and 3.
- Descending order and backend pagination are implemented in Task 3 and exercised in Task 5.
- Mihomo rule injection is implemented in Task 4.

## Placeholder Scan

- No `TODO`, `TBD`, or deferred “implement later” placeholders remain.
- Each task lists exact file paths, code snippets, and verification commands.
- The only assumptions are captured explicitly at the top of the plan.

## Type Consistency Check

- API JSON uses `rule_type`, `pattern`, and `proxy_group` consistently across model, handler, tests, and client.
- Pagination JSON uses `page`, `page_size`, and `total` consistently across handler, tests, and client.
- The renderer signature is updated consistently to `MihomoTemplate(templatePath, nodes, manualRules)`.

