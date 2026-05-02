# Phase 3 Proxy Group Backend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add proxy group CRUD plus on-demand group node resolution, where script text is stored in the database, membership is never stored, empty scripts include all nodes, and JavaScript script failures fall back to all nodes.

**Architecture:** Introduce a dedicated `group` package that owns proxy group persistence, HTTP CRUD, and on-demand node resolution. Persist only group metadata and script text in a new `proxy_groups` table; do not add any join table for group-to-node membership. Resolve group nodes by querying the current `proxy_nodes` set at read time, and when a group has a script, evaluate it with `goja` against `ProxyNodeView` values shaped as `{id, providerName, name}`.

**Tech Stack:** Go, `database/sql`, `modernc.org/sqlite`, `net/http`, `github.com/dop251/goja`, `stretchr/testify`

---

## Assumptions

- The script return value `number[]` represents `ProxyNodeView.id` values, not positional indexes.
- Returning any non-array value, non-numeric value, duplicate id, or unknown id is treated as script failure and triggers the full-node fallback.
- Phase 3 backend scope stops at CRUD plus computed node reads; Mihomo template output is unchanged in this plan.

## File Structure

**Modify**

- `go.mod`
- `go.sum`
- `main.go`
- `internal/store/migrations/001_initial.sql`

**Create**

- `internal/group/model.go`
- `internal/group/repository.go`
- `internal/group/service.go`
- `internal/group/http.go`
- `internal/group/script.go`
- `tests/integration/group_api_test.go`
- `tests/unit/group_script_test.go`

**Responsibility Notes**

- `internal/store/migrations/001_initial.sql`: add a `proxy_groups` table only. Do not add any membership table.
- `internal/group/model.go`: define API-facing group models plus the script input shape `ProxyNodeView`.
- `internal/group/repository.go`: own proxy group CRUD and the SQL query that computes the current `ProxyNodeView` list from `proxy_nodes` joined with `providers`.
- `internal/group/service.go`: validate group input, keep scripts optional, orchestrate on-demand node resolution, and apply the fallback-to-all behavior.
- `internal/group/script.go`: compile and run JavaScript functions with `goja`, enforcing the function-only and `number[]` contract.
- `internal/group/http.go`: expose `/proxy-groups` CRUD plus `GET /proxy-groups/{id}/nodes` using the same manual path parsing style as `provider/http.go`.
- `main.go`: wire the new group repository, service, and handler into the existing HTTP server.
- `tests/integration/group_api_test.go`: cover CRUD, optional scripts, computed membership, and fallback behavior through real HTTP plus real SQLite.
- `tests/unit/group_script_test.go`: lock down the JavaScript runner contract with focused, fast tests.

## Task 1: Add the Proxy Group Schema and Wire the New Routes

**Files:**

- Modify: `internal/store/migrations/001_initial.sql`
- Modify: `main.go`
- Create: `internal/group/model.go`
- Create: `internal/group/repository.go`
- Create: `internal/group/service.go`
- Create: `internal/group/http.go`
- Test: `tests/integration/group_api_test.go`

- [ ] **Step 1: Write a failing integration test for the empty list route**

```go
func TestListProxyGroupsStartsEmpty(t *testing.T) {
	ts, _ := newTestServerWithRefresh(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/proxy-groups")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.JSONEq(t, `{"groups":[]}`, readBody(t, resp))
}
```

- [ ] **Step 2: Run the focused integration test to confirm the route does not exist yet**

Run: `go test ./tests/integration -run TestListProxyGroupsStartsEmpty -v`
Expected: FAIL with `404` or missing route behavior because `/proxy-groups` is not wired yet.

- [ ] **Step 3: Add the schema, model, repository, service, and handler skeleton**

```sql
CREATE TABLE IF NOT EXISTS proxy_groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    script_text TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
```

```go
type ProxyGroup struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Script    string    `json:"script"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ProxyNodeView struct {
	ID           int64  `json:"id"`
	ProviderName string `json:"providerName"`
	Name         string `json:"name"`
}
```

```go
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/proxy-groups", h.handleGroups)
	mux.HandleFunc("/proxy-groups/", h.handleGroupByID)
}
```

```go
groupRepo := group.NewRepository(db)
groupSvc := group.NewService(groupRepo)
groupHandler := group.NewHandler(groupSvc)
groupHandler.RegisterRoutes(mux)
```

- [ ] **Step 4: Re-run the focused integration test and confirm the empty list route now passes**

Run: `go test ./tests/integration -run TestListProxyGroupsStartsEmpty -v`
Expected: PASS

- [ ] **Step 5: Commit the schema and route baseline**

```bash
git add internal/store/migrations/001_initial.sql internal/group/model.go internal/group/repository.go internal/group/service.go internal/group/http.go main.go tests/integration/group_api_test.go
git commit -m "feat: add proxy group schema and routes"
```

## Task 2: Implement Proxy Group CRUD with Script Text Persistence

**Files:**

- Modify: `internal/group/repository.go`
- Modify: `internal/group/service.go`
- Modify: `internal/group/http.go`
- Test: `tests/integration/group_api_test.go`

- [ ] **Step 1: Write failing CRUD integration tests that prove script text is stored in the database-backed API**

```go
func TestCreateGetUpdateDeleteProxyGroup(t *testing.T) {
	ts, _ := newTestServerWithRefresh(t)
	defer ts.Close()

	createResp := postJSON(t, ts.URL+"/proxy-groups", `{"name":"OpenAI","script":"function (proxyNodes) { return proxyNodes.map(function (node) { return node.id }) }"}`)
	defer createResp.Body.Close()
	require.Equal(t, http.StatusCreated, createResp.StatusCode)

	var created struct {
		Group struct {
			ID     int64  `json:"id"`
			Name   string `json:"name"`
			Script string `json:"script"`
		} `json:"group"`
	}
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&created))
	assert.Equal(t, "OpenAI", created.Group.Name)
	assert.Contains(t, created.Group.Script, "return node.id")

	getResp, err := http.Get(fmt.Sprintf("%s/proxy-groups/%d", ts.URL, created.Group.ID))
	require.NoError(t, err)
	defer getResp.Body.Close()
	assert.Equal(t, http.StatusOK, getResp.StatusCode)
	assert.Contains(t, readBody(t, getResp), `"name":"OpenAI"`)

	updateResp := putJSON(t, fmt.Sprintf("%s/proxy-groups/%d", ts.URL, created.Group.ID), `{"name":"Streaming","script":""}`)
	defer updateResp.Body.Close()
	assert.Equal(t, http.StatusOK, updateResp.StatusCode)
	assert.Contains(t, readBody(t, updateResp), `"script":""`)

	deleteResp := deleteRequest(t, fmt.Sprintf("%s/proxy-groups/%d", ts.URL, created.Group.ID))
	defer deleteResp.Body.Close()
	assert.Equal(t, http.StatusNoContent, deleteResp.StatusCode)
}
```

- [ ] **Step 2: Run the CRUD test to capture the missing behavior**

Run: `go test ./tests/integration -run TestCreateGetUpdateDeleteProxyGroup -v`
Expected: FAIL because the CRUD repository, service, and handler methods are still stubs or incomplete.

- [ ] **Step 3: Implement repository CRUD methods and keep `script_text` as the persisted source of truth**

```go
func (r *Repository) Create(ctx context.Context, g ProxyGroup) (ProxyGroup, error) {
	now := nowInLocation()
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO proxy_groups (name, script_text, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		g.Name, g.Script, now.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	if err != nil {
		return ProxyGroup{}, err
	}
	id, _ := result.LastInsertId()
	g.ID = id
	g.CreatedAt = now
	g.UpdatedAt = now
	return g, nil
}
```

```go
type CreateGroupInput struct {
	Name   string `json:"name"`
	Script string `json:"script"`
}

type UpdateGroupInput struct {
	Name   string `json:"name"`
	Script string `json:"script"`
}
```

```go
switch r.Method {
case http.MethodGet:
	h.listGroups(w, r)
case http.MethodPost:
	h.createGroup(w, r)
default:
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}
```

- [ ] **Step 4: Re-run the CRUD test until create, get, update, and delete pass**

Run: `go test ./tests/integration -run TestCreateGetUpdateDeleteProxyGroup -v`
Expected: PASS

- [ ] **Step 5: Commit the CRUD slice**

```bash
git add internal/group/repository.go internal/group/service.go internal/group/http.go tests/integration/group_api_test.go
git commit -m "feat: add proxy group crud"
```

## Task 3: Compute Group Nodes On the Fly and Return All Nodes When Script Is Empty

**Files:**

- Modify: `internal/group/repository.go`
- Modify: `internal/group/service.go`
- Modify: `internal/group/http.go`
- Test: `tests/integration/group_api_test.go`

- [ ] **Step 1: Write a failing integration test for the no-script default behavior**

```go
func TestProxyGroupWithoutScriptReturnsAllProxyNodes(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "fixtures", "provider_plain.yaml"))
	require.NoError(t, err)

	upstream := newFlakyUpstream(t, fixture)
	ts, _ := newTestServerWithRefresh(t)
	defer ts.Close()

	providerID := createProvider(t, ts.URL, fmt.Sprintf(`{"name":"alpha","url":"%s"}`, upstream.server.URL))
	refreshResp := refreshProvider(t, ts.URL, providerID)
	defer refreshResp.Body.Close()
	require.Equal(t, http.StatusNoContent, refreshResp.StatusCode)

	createResp := postJSON(t, ts.URL+"/proxy-groups", `{"name":"All Nodes","script":""}`)
	defer createResp.Body.Close()
	require.Equal(t, http.StatusCreated, createResp.StatusCode)

	var created struct {
		Group struct {
			ID int64 `json:"id"`
		} `json:"group"`
	}
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&created))

	nodesResp, err := http.Get(fmt.Sprintf("%s/proxy-groups/%d/nodes", ts.URL, created.Group.ID))
	require.NoError(t, err)
	defer nodesResp.Body.Close()
	require.Equal(t, http.StatusOK, nodesResp.StatusCode)

	var result struct {
		Nodes []struct {
			ID           int64  `json:"id"`
			ProviderName string `json:"providerName"`
			Name         string `json:"name"`
		} `json:"nodes"`
	}
	require.NoError(t, json.NewDecoder(nodesResp.Body).Decode(&result))
	assert.Len(t, result.Nodes, 2)
	assert.Equal(t, "alpha", result.Nodes[0].ProviderName)
	assert.Equal(t, "vmess-hk-01", result.Nodes[0].Name)
	assert.Equal(t, "ss-jp-01", result.Nodes[1].Name)
}
```

- [ ] **Step 2: Run the test and confirm the computed node route is missing or incomplete**

Run: `go test ./tests/integration -run TestProxyGroupWithoutScriptReturnsAllProxyNodes -v`
Expected: FAIL because `/proxy-groups/{id}/nodes` does not resolve current proxy nodes yet.

- [ ] **Step 3: Implement the repository query and service fallback for scriptless groups**

```go
func (r *Repository) ListProxyNodeViews(ctx context.Context) ([]ProxyNodeView, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT n.id, p.name, n.name
		 FROM proxy_nodes n
		 JOIN providers p ON p.id = n.provider_id
		 ORDER BY p.id, n.id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []ProxyNodeView
	for rows.Next() {
		var node ProxyNodeView
		if err := rows.Scan(&node.ID, &node.ProviderName, &node.Name); err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	return nodes, rows.Err()
}
```

```go
func (s *Service) ListNodes(ctx context.Context, groupID int64) ([]ProxyNodeView, error) {
	group, err := s.repo.GetByID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	nodes, err := s.repo.ListProxyNodeViews(ctx)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(group.Script) == "" {
		return nodes, nil
	}
	return s.selectNodes(group.Script, nodes)
}
```

```go
case "nodes":
	if r.Method == http.MethodGet {
		h.getGroupNodes(w, r, id)
		return
	}
```

- [ ] **Step 4: Re-run the no-script integration test until it passes**

Run: `go test ./tests/integration -run TestProxyGroupWithoutScriptReturnsAllProxyNodes -v`
Expected: PASS

- [ ] **Step 5: Commit the on-demand default behavior**

```bash
git add internal/group/repository.go internal/group/service.go internal/group/http.go tests/integration/group_api_test.go
git commit -m "feat: compute group nodes without membership storage"
```

## Task 4: Add the JavaScript Runner with the Function-Only Contract

**Files:**

- Modify: `go.mod`
- Modify: `go.sum`
- Create: `internal/group/script.go`
- Test: `tests/unit/group_script_test.go`

- [ ] **Step 1: Write failing unit tests for the runner contract**

```go
func TestSelectNodeIDsReturnsAllNodesForEmptyScript(t *testing.T) {
	nodes := []group.ProxyNodeView{
		{ID: 11, ProviderName: "alpha", Name: "vmess-hk-01"},
		{ID: 12, ProviderName: "alpha", Name: "ss-jp-01"},
	}

	selected, err := group.SelectNodeIDs("", nodes)
	require.NoError(t, err)
	assert.Equal(t, []int64{11, 12}, selected)
}

func TestSelectNodeIDsRunsJavaScriptFunction(t *testing.T) {
	nodes := []group.ProxyNodeView{
		{ID: 11, ProviderName: "alpha", Name: "vmess-hk-01"},
		{ID: 12, ProviderName: "beta", Name: "ss-jp-01"},
	}

	selected, err := group.SelectNodeIDs(`function (proxyNodes) { return [proxyNodes[1].id] }`, nodes)
	require.NoError(t, err)
	assert.Equal(t, []int64{12}, selected)
}

func TestSelectNodeIDsRejectsNonFunctionScript(t *testing.T) {
	nodes := []group.ProxyNodeView{{ID: 11, ProviderName: "alpha", Name: "vmess-hk-01"}}

	_, err := group.SelectNodeIDs(`var x = 1`, nodes)
	require.Error(t, err)
}

func TestSelectNodeIDsRejectsUnknownIDs(t *testing.T) {
	nodes := []group.ProxyNodeView{{ID: 11, ProviderName: "alpha", Name: "vmess-hk-01"}}

	_, err := group.SelectNodeIDs(`function (proxyNodes) { return [999] }`, nodes)
	require.Error(t, err)
}
```

- [ ] **Step 2: Run the unit tests to verify the runner is missing**

Run: `go test ./tests/unit -run 'TestSelectNodeIDs' -v`
Expected: FAIL because `SelectNodeIDs` and the `goja` integration do not exist yet.

- [ ] **Step 3: Add `goja` and implement the function-only runner**

```bash
go get github.com/dop251/goja
```

```go
func SelectNodeIDs(script string, nodes []ProxyNodeView) ([]int64, error) {
	if strings.TrimSpace(script) == "" {
		return allNodeIDs(nodes), nil
	}

	vm := goja.New()
	value, err := vm.RunString("(" + script + ")")
	if err != nil {
		return nil, err
	}

	fn, ok := goja.AssertFunction(value)
	if !ok {
		return nil, errors.New("script must evaluate to a function")
	}

	arg := vm.ToValue(nodes)
	result, err := fn(goja.Undefined(), arg)
	if err != nil {
		return nil, err
	}

	return exportNodeIDs(result.Export(), nodes)
}
```

```go
func exportNodeIDs(value any, nodes []ProxyNodeView) ([]int64, error) {
	raw, ok := value.([]any)
	if !ok {
		return nil, errors.New("script must return number[]")
	}

	allowed := map[int64]struct{}{}
	for _, node := range nodes {
		allowed[node.ID] = struct{}{}
	}

	var ids []int64
	seen := map[int64]struct{}{}
	for _, item := range raw {
		number, ok := item.(int64)
		if !ok {
			floatNumber, floatOK := item.(float64)
			if !floatOK {
				return nil, errors.New("script must return number[]")
			}
			number = int64(floatNumber)
			if float64(number) != floatNumber {
				return nil, errors.New("script must return whole-number ids")
			}
		}
		if _, ok := allowed[number]; !ok {
			return nil, errors.New("script returned unknown proxy node id")
		}
		if _, ok := seen[number]; ok {
			return nil, errors.New("script returned duplicate proxy node id")
		}
		seen[number] = struct{}{}
		ids = append(ids, number)
	}
	return ids, nil
}
```

- [ ] **Step 4: Re-run the unit tests until the runner contract passes**

Run: `go test ./tests/unit -run 'TestSelectNodeIDs' -v`
Expected: PASS

- [ ] **Step 5: Commit the script runner**

```bash
git add go.mod go.sum internal/group/script.go tests/unit/group_script_test.go
git commit -m "feat: add proxy group javascript runner"
```

## Task 5: Apply Scripted Selection and Fall Back to All Nodes on Script Error

**Files:**

- Modify: `internal/group/service.go`
- Modify: `tests/integration/group_api_test.go`

- [ ] **Step 1: Write failing integration tests for scripted filtering and error fallback**

```go
func TestProxyGroupScriptFiltersNodesByReturnedIDs(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "fixtures", "provider_plain.yaml"))
	require.NoError(t, err)

	upstream := newFlakyUpstream(t, fixture)
	ts, _ := newTestServerWithRefresh(t)
	defer ts.Close()

	providerID := createProvider(t, ts.URL, fmt.Sprintf(`{"name":"alpha","url":"%s"}`, upstream.server.URL))
	refreshResp := refreshProvider(t, ts.URL, providerID)
	defer refreshResp.Body.Close()
	require.Equal(t, http.StatusNoContent, refreshResp.StatusCode)

	createResp := postJSON(t, ts.URL+"/proxy-groups", `{"name":"Filtered","script":"function (proxyNodes) { return [proxyNodes[1].id] }"}`)
	defer createResp.Body.Close()
	require.Equal(t, http.StatusCreated, createResp.StatusCode)

	var created struct {
		Group struct {
			ID int64 `json:"id"`
		} `json:"group"`
	}
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&created))

	nodesResp, err := http.Get(fmt.Sprintf("%s/proxy-groups/%d/nodes", ts.URL, created.Group.ID))
	require.NoError(t, err)
	defer nodesResp.Body.Close()

	var result struct {
		Nodes []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"nodes"`
	}
	require.NoError(t, json.NewDecoder(nodesResp.Body).Decode(&result))
	require.Len(t, result.Nodes, 1)
	assert.Equal(t, "ss-jp-01", result.Nodes[0].Name)
}

func TestProxyGroupScriptErrorFallsBackToAllProxyNodes(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "fixtures", "provider_plain.yaml"))
	require.NoError(t, err)

	upstream := newFlakyUpstream(t, fixture)
	ts, _ := newTestServerWithRefresh(t)
	defer ts.Close()

	providerID := createProvider(t, ts.URL, fmt.Sprintf(`{"name":"alpha","url":"%s"}`, upstream.server.URL))
	refreshResp := refreshProvider(t, ts.URL, providerID)
	defer refreshResp.Body.Close()
	require.Equal(t, http.StatusNoContent, refreshResp.StatusCode)

	createResp := postJSON(t, ts.URL+"/proxy-groups", `{"name":"Broken","script":"function (proxyNodes) { throw new Error('boom') }"}`)
	defer createResp.Body.Close()
	require.Equal(t, http.StatusCreated, createResp.StatusCode)

	var created struct {
		Group struct {
			ID int64 `json:"id"`
		} `json:"group"`
	}
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&created))

	nodesResp, err := http.Get(fmt.Sprintf("%s/proxy-groups/%d/nodes", ts.URL, created.Group.ID))
	require.NoError(t, err)
	defer nodesResp.Body.Close()

	var result struct {
		Nodes []struct {
			ID int64 `json:"id"`
		} `json:"nodes"`
	}
	require.NoError(t, json.NewDecoder(nodesResp.Body).Decode(&result))
	assert.Len(t, result.Nodes, 2)
}
```

- [ ] **Step 2: Run the integration tests and capture the missing script behavior**

Run: `go test ./tests/integration -run 'TestProxyGroupScriptFiltersNodesByReturnedIDs|TestProxyGroupScriptErrorFallsBackToAllProxyNodes' -v`
Expected: FAIL because node resolution does not apply the JavaScript runner yet.

- [ ] **Step 3: Implement script application with fallback-to-all on any runner error**

```go
func (s *Service) selectNodes(script string, nodes []ProxyNodeView) ([]ProxyNodeView, error) {
	selectedIDs, err := SelectNodeIDs(script, nodes)
	if err != nil {
		return nodes, nil
	}

	byID := map[int64]ProxyNodeView{}
	for _, node := range nodes {
		byID[node.ID] = node
	}

	var selected []ProxyNodeView
	for _, id := range selectedIDs {
		selected = append(selected, byID[id])
	}
	return selected, nil
}
```

```go
func (s *Service) ListNodes(ctx context.Context, groupID int64) ([]ProxyNodeView, error) {
	group, err := s.repo.GetByID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	nodes, err := s.repo.ListProxyNodeViews(ctx)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(group.Script) == "" {
		return nodes, nil
	}
	return s.selectNodes(group.Script, nodes)
}
```

- [ ] **Step 4: Re-run the scripted integration tests until both filtering and fallback behavior pass**

Run: `go test ./tests/integration -run 'TestProxyGroupScriptFiltersNodesByReturnedIDs|TestProxyGroupScriptErrorFallsBackToAllProxyNodes' -v`
Expected: PASS

- [ ] **Step 5: Commit the scripted selection behavior**

```bash
git add internal/group/service.go tests/integration/group_api_test.go
git commit -m "feat: apply proxy group scripts with fallback"
```

## Task 6: Prove Per-Group Isolation and Script Removal Through API Behavior

**Files:**

- Modify: `tests/integration/group_api_test.go`
- Modify: `internal/group/service.go`

- [ ] **Step 1: Write failing integration tests for per-group isolation and script removal**

```go
func TestUpdatingOneGroupScriptDoesNotChangeOtherGroups(t *testing.T) {
	ts, _ := newTestServerWithRefresh(t)
	defer ts.Close()

	firstResp := postJSON(t, ts.URL+"/proxy-groups", `{"name":"First","script":"function (proxyNodes) { return [] }"}`)
	defer firstResp.Body.Close()
	secondResp := postJSON(t, ts.URL+"/proxy-groups", `{"name":"Second","script":"function (proxyNodes) { return proxyNodes.map(function (node) { return node.id }) }"}`)
	defer secondResp.Body.Close()

	var first struct {
		Group struct {
			ID int64 `json:"id"`
		} `json:"group"`
	}
	var second struct {
		Group struct {
			ID int64 `json:"id"`
		} `json:"group"`
	}
	require.NoError(t, json.NewDecoder(firstResp.Body).Decode(&first))
	require.NoError(t, json.NewDecoder(secondResp.Body).Decode(&second))

	updateResp := putJSON(t, fmt.Sprintf("%s/proxy-groups/%d", ts.URL, first.Group.ID), `{"name":"First","script":""}`)
	defer updateResp.Body.Close()
	require.Equal(t, http.StatusOK, updateResp.StatusCode)

	getResp, err := http.Get(fmt.Sprintf("%s/proxy-groups/%d", ts.URL, second.Group.ID))
	require.NoError(t, err)
	defer getResp.Body.Close()
	assert.Contains(t, readBody(t, getResp), "return node.id")
}

func TestRemovingScriptKeepsGroupValid(t *testing.T) {
	ts, _ := newTestServerWithRefresh(t)
	defer ts.Close()

	createResp := postJSON(t, ts.URL+"/proxy-groups", `{"name":"Optional","script":"function (proxyNodes) { return [] }"}`)
	defer createResp.Body.Close()

	var created struct {
		Group struct {
			ID int64 `json:"id"`
		} `json:"group"`
	}
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&created))

	updateResp := putJSON(t, fmt.Sprintf("%s/proxy-groups/%d", ts.URL, created.Group.ID), `{"name":"Optional","script":""}`)
	defer updateResp.Body.Close()
	require.Equal(t, http.StatusOK, updateResp.StatusCode)

	getResp, err := http.Get(fmt.Sprintf("%s/proxy-groups/%d", ts.URL, created.Group.ID))
	require.NoError(t, err)
	defer getResp.Body.Close()
	assert.Equal(t, http.StatusOK, getResp.StatusCode)
	assert.Contains(t, readBody(t, getResp), `"script":""`)
}
```

- [ ] **Step 2: Run the isolation tests to catch any accidental shared-state behavior**

Run: `go test ./tests/integration -run 'TestUpdatingOneGroupScriptDoesNotChangeOtherGroups|TestRemovingScriptKeepsGroupValid' -v`
Expected: FAIL if update logic mutates shared state or if empty script is rejected.

- [ ] **Step 3: Tighten service update behavior so each group is updated independently and empty script remains valid**

```go
func (s *Service) Update(ctx context.Context, id int64, in UpdateGroupInput) (ProxyGroup, error) {
	group, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return ProxyGroup{}, err
	}
	group.Name = strings.TrimSpace(in.Name)
	group.Script = strings.TrimSpace(in.Script)
	return s.repo.Update(ctx, group)
}
```

- [ ] **Step 4: Re-run the isolation tests until they pass**

Run: `go test ./tests/integration -run 'TestUpdatingOneGroupScriptDoesNotChangeOtherGroups|TestRemovingScriptKeepsGroupValid' -v`
Expected: PASS

- [ ] **Step 5: Commit the optional-script and isolation guarantees**

```bash
git add internal/group/service.go tests/integration/group_api_test.go
git commit -m "feat: preserve per-group script isolation"
```

## Task 7: Final Verification

**Files:**

- Modify: none
- Test: `tests/integration/group_api_test.go`
- Test: `tests/unit/group_script_test.go`

- [ ] **Step 1: Run the focused proxy group test suites**

Run: `go test ./tests/integration -run 'TestListProxyGroupsStartsEmpty|TestCreateGetUpdateDeleteProxyGroup|TestProxyGroupWithoutScriptReturnsAllProxyNodes|TestProxyGroupScriptFiltersNodesByReturnedIDs|TestProxyGroupScriptErrorFallsBackToAllProxyNodes|TestUpdatingOneGroupScriptDoesNotChangeOtherGroups|TestRemovingScriptKeepsGroupValid' -v`
Expected: PASS

- [ ] **Step 2: Run the unit test suite for the JavaScript runner**

Run: `go test ./tests/unit -run 'TestSelectNodeIDs' -v`
Expected: PASS

- [ ] **Step 3: Run the full repository test suite**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 4: Run static analysis**

Run: `go vet ./...`
Expected: PASS

- [ ] **Step 5: Commit the verified Phase 3 backend slice**

```bash
git add go.mod go.sum main.go internal/store/migrations/001_initial.sql internal/group/model.go internal/group/repository.go internal/group/service.go internal/group/http.go internal/group/script.go tests/integration/group_api_test.go tests/unit/group_script_test.go
git commit -m "feat: add proxy group backend"
```
