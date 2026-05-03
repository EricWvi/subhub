# Phase 4 Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refactor Clash config subscriptions so output-facing proxy groups have explicit persisted ordering, the reserved `Proxies` group is fixed at position `0` and non-deletable but still editable for members and rule binding, and all related backend, rendering, and UI logic stays consistent.

**Architecture:** Keep the existing `subscription` package and extend it rather than redesigning Phase 4. Add a dedicated `position` column to `clash_config_proxy_groups`, backfill existing rows in migration `004`, and make repository reads/writes order by `position` instead of `id`. Treat `Proxies` as a reserved persisted row with special rules only for deletion and ordering; otherwise it follows the same membership and binding model as other output-facing proxy groups.

**Tech Stack:** Go, `database/sql`, `modernc.org/sqlite`, embedded SQL migrations, `net/http`, React 19, Ant Design, TypeScript, `stretchr/testify`

---

## Confirmed Business Rules

- `clash_config_proxy_groups` needs a persisted `position` column because subscription output order matters.
- Existing subscriptions should keep their current order by backfilling `position` from current `id` order in migration `004`.
- The reserved `Proxies` proxy group always exists at position `0`.
- The reserved `Proxies` proxy group is non-deletable.
- The reserved `Proxies` proxy group is editable for:
  - its member list
  - its bound internal proxy group
- The reserved `Proxies` proxy group keeps the current default initial members: `DIRECT` only.
- The reserved `Proxies` proxy group has the same member-type scope as other output-facing proxy groups.

## File Structure

**Modify**

- `internal/store/migrations/003_add_subscriptions.sql`
- `internal/subscription/model.go`
- `internal/subscription/repository.go`
- `internal/subscription/service.go`
- `client/src/components/ClashConfigSubscriptionManager.tsx`
- `tests/integration/subscription_api_test.go`
- `tests/integration/subscription_output_test.go`

**Create**

- `internal/store/migrations/004_add_clash_config_proxy_group_position.sql`

## Responsibility Notes

- `internal/store/migrations/004_add_clash_config_proxy_group_position.sql`: add and backfill `position` for existing `clash_config_proxy_groups` rows.
- `internal/subscription/model.go`: expose `position` on Clash config proxy groups and on create/update input payloads if the API persists whole ordered lists.
- `internal/subscription/repository.go`: persist proxy-group order on create/update and load groups ordered by `position`.
- `internal/subscription/service.go`: normalize the reserved `Proxies` row so it stays at `0`, cannot be deleted, and remains editable only in the allowed ways.
- `client/src/components/ClashConfigSubscriptionManager.tsx`: always show `Proxies` in the form, prevent deleting it, keep it visually first, and allow editing members plus rule binding.
- tests: lock migration behavior, CRUD behavior, ordering behavior, and output order.

## Task 1: Add Persisted Proxy Group Ordering

**Files:**

- Create: `internal/store/migrations/004_add_clash_config_proxy_group_position.sql`
- Modify: `internal/subscription/model.go`
- Modify: `internal/subscription/repository.go`
- Test: `tests/integration/subscription_api_test.go`

- [ ] **Step 1: Add a failing integration test that proves proxy-group order must round-trip**

```go
func TestClashConfigSubscriptionProxyGroupsPreserveExplicitOrder(t *testing.T) {
	ts := newTestServerWithSubscriptions(t)
	defer ts.Close()

	providerID := createProvider(t, ts.URL, `{"name":"alpha","url":"https://example.com/a"}`)
	groupA := createProxyGroup(t, ts.URL, `{"name":"A","script":""}`)
	groupB := createProxyGroup(t, ts.URL, `{"name":"B","script":""}`)
	groupC := createProxyGroup(t, ts.URL, `{"name":"C","script":""}`)

	resp := postJSON(t, ts.URL+"/api/subscriptions/clash-configs", fmt.Sprintf(`{
		"name":"Daily",
		"providers":[%d],
		"proxy_groups":[
			{
				"name":"Proxies",
				"type":"select",
				"position":0,
				"proxies":[{"type":"DIRECT","value":"DIRECT"}],
				"bind_internal_proxy_group_id":%d
			},
			{
				"name":"Fallback",
				"type":"fallback",
				"position":2,
				"url":"https://cp.cloudflare.com/generate_204",
				"interval":300,
				"proxies":[{"type":"internal","value":"%d"}],
				"bind_internal_proxy_group_id":%d
			},
			{
				"name":"Auto",
				"type":"url-test",
				"position":1,
				"url":"https://cp.cloudflare.com/generate_204",
				"interval":300,
				"proxies":[{"type":"internal","value":"%d"}],
				"bind_internal_proxy_group_id":%d
			}
		]
	}`, providerID, groupA, groupB, groupB, groupC, groupC))
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode)
	body := readBody(t, resp)
	assert.Regexp(t, `"proxy_groups":\[\{"id":[0-9]+,"name":"Proxies".*"position":0`, body)
	assert.True(t, strings.Index(body, `"name":"Proxies"`) < strings.Index(body, `"name":"Auto"`))
	assert.True(t, strings.Index(body, `"name":"Auto"`) < strings.Index(body, `"name":"Fallback"`))
}
```

- [ ] **Step 2: Run the focused test to confirm current code still orders by `id`**

Run: `go test ./tests/integration -run TestClashConfigSubscriptionProxyGroupsPreserveExplicitOrder -v`
Expected: FAIL because `clash_config_proxy_groups` has no `position` column and repository reads still order by `id`.

- [ ] **Step 3: Add migration `004` and wire `position` through models and repository persistence**

```sql
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
```

```go
type ClashConfigProxyGroup struct {
	ID                  int64         `json:"id"`
	Name                string        `json:"name"`
	Type                string        `json:"type"`
	Position            int64         `json:"position"`
	URL                 string        `json:"url"`
	Interval            int64         `json:"interval"`
	Proxies             []ProxyMember `json:"proxies"`
	BindInternalGroupID int64         `json:"bind_internal_proxy_group_id"`
	IsSystem            bool          `json:"is_system"`
}

type CreateClashConfigProxyGroupInput struct {
	Name                string        `json:"name"`
	Type                string        `json:"type"`
	Position            int64         `json:"position"`
	URL                 string        `json:"url"`
	Interval            int64         `json:"interval"`
	Proxies             []ProxyMember `json:"proxies"`
	BindInternalGroupID int64         `json:"bind_internal_proxy_group_id"`
}
```

```go
result, err := tx.ExecContext(ctx,
	`INSERT INTO clash_config_proxy_groups
	 (subscription_id, name, type, position, url, interval_seconds, bind_internal_proxy_group_id, is_system, created_at, updated_at)
	 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
	subID, pg.Name, pg.Type, pg.Position, pg.URL, pg.Interval, pg.BindInternalGroupID, isSystem, nowStr, nowStr,
)
```

```go
rows, err := r.db.QueryContext(ctx,
	`SELECT id, name, type, position, url, interval_seconds, bind_internal_proxy_group_id, is_system
	 FROM clash_config_proxy_groups
	 WHERE subscription_id = ?
	 ORDER BY position, id`, subID,
)
```

- [ ] **Step 4: Re-run the focused order test and the migration regression test**

Run: `go test ./tests/integration -run 'TestClashConfigSubscriptionProxyGroupsPreserveExplicitOrder|TestStoreAppliesAllMigrations' -v`
Expected: PASS

- [ ] **Step 5: Commit the persisted proxy-group ordering work**

```bash
git add internal/store/migrations/004_add_clash_config_proxy_group_position.sql internal/subscription/model.go internal/subscription/repository.go tests/integration/subscription_api_test.go
git commit -m "refactor: persist clash config proxy group order"
```

## Task 2: Refactor Reserved `Proxies` Behavior

**Files:**

- Modify: `internal/subscription/service.go`
- Modify: `internal/subscription/repository.go`
- Modify: `internal/subscription/model.go`
- Test: `tests/integration/subscription_api_test.go`

- [ ] **Step 1: Add failing tests for the reserved `Proxies` row rules**

```go
func TestCreateClashConfigSubscriptionRequiresReservedProxiesAtPositionZero(t *testing.T) {
	ts := newTestServerWithSubscriptions(t)
	defer ts.Close()

	providerID := createProvider(t, ts.URL, `{"name":"alpha","url":"https://example.com/a"}`)
	groupID := createProxyGroup(t, ts.URL, `{"name":"Streaming","script":""}`)

	resp := postJSON(t, ts.URL+"/api/subscriptions/clash-configs", fmt.Sprintf(`{
		"name":"Daily",
		"providers":[%d],
		"proxy_groups":[
			{
				"name":"Media",
				"type":"select",
				"position":0,
				"proxies":[{"type":"DIRECT","value":"DIRECT"}],
				"bind_internal_proxy_group_id":%d
			}
		]
	}`, providerID, groupID))
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, readBody(t, resp), "Proxies")
}

func TestUpdateClashConfigSubscriptionAllowsEditingReservedProxiesMembersAndBinding(t *testing.T) {
	ts := newTestServerWithSubscriptions(t)
	defer ts.Close()

	providerID := createProvider(t, ts.URL, `{"name":"alpha","url":"https://example.com/a"}`)
	groupA := createProxyGroup(t, ts.URL, `{"name":"A","script":""}`)
	groupB := createProxyGroup(t, ts.URL, `{"name":"B","script":""}`)

	createResp := postJSON(t, ts.URL+"/api/subscriptions/clash-configs", fmt.Sprintf(`{
		"name":"Daily",
		"providers":[%d],
		"proxy_groups":[
			{
				"name":"Proxies",
				"type":"select",
				"position":0,
				"proxies":[{"type":"DIRECT","value":"DIRECT"}],
				"bind_internal_proxy_group_id":%d
			}
		]
	}`, providerID, groupA))
	defer createResp.Body.Close()
	require.Equal(t, http.StatusCreated, createResp.StatusCode)

	updateResp := putJSON(t, ts.URL+"/api/subscriptions/clash-configs/1", fmt.Sprintf(`{
		"name":"Daily",
		"providers":[%d],
		"proxy_groups":[
			{
				"name":"Proxies",
				"type":"select",
				"position":0,
				"proxies":[
					{"type":"reference","value":"Proxies"},
					{"type":"internal","value":"%d"},
					{"type":"DIRECT","value":"DIRECT"}
				],
				"bind_internal_proxy_group_id":%d
			}
		]
	}`, providerID, groupB, groupB))
	defer updateResp.Body.Close()

	assert.Equal(t, http.StatusOK, updateResp.StatusCode)
	body := readBody(t, updateResp)
	assert.Contains(t, body, `"name":"Proxies"`)
	assert.Contains(t, body, `"bind_internal_proxy_group_id":2`)
	assert.Contains(t, body, `"type":"reference","value":"Proxies"`)
}
```

- [ ] **Step 2: Run the focused tests to capture the reserved-row mismatch**

Run: `go test ./tests/integration -run 'TestCreateClashConfigSubscriptionRequiresReservedProxiesAtPositionZero|TestUpdateClashConfigSubscriptionAllowsEditingReservedProxiesMembersAndBinding' -v`
Expected: FAIL because the current service auto-creates `Proxies` separately and does not treat it as part of the editable ordered payload.

- [ ] **Step 3: Refactor service validation so `Proxies` is a required reserved row in the payload**

```go
var (
	ErrReservedProxyGroupMissing      = errors.New("Proxies proxy-group is required")
	ErrReservedProxyGroupWrongOrder   = errors.New("Proxies proxy-group must stay at position 0")
	ErrReservedProxyGroupDelete       = errors.New("Proxies proxy-group cannot be deleted")
	ErrReservedProxyGroupImmutableKey = errors.New("Proxies proxy-group name and type are fixed")
)
```

```go
func normalizeClashConfigProxyGroups(in []CreateClashConfigProxyGroupInput) ([]CreateClashConfigProxyGroupInput, error) {
	if len(in) == 0 {
		return nil, ErrReservedProxyGroupMissing
	}

	seenProxies := false
	for i := range in {
		if in[i].Name != "Proxies" {
			continue
		}
		if in[i].Type != "select" {
			return nil, ErrReservedProxyGroupImmutableKey
		}
		if in[i].Position != 0 || i != 0 {
			return nil, ErrReservedProxyGroupWrongOrder
		}
		seenProxies = true
	}
	if !seenProxies {
		return nil, ErrReservedProxyGroupMissing
	}

	for i := range in {
		in[i].Position = int64(i)
	}
	return in, nil
}
```

```go
func (s *Service) CreateClashConfig(ctx context.Context, in CreateClashConfigSubscriptionInput) (ClashConfigSubscription, error) {
	if in.Name == "" {
		return ClashConfigSubscription{}, ErrSubscriptionNameRequired
	}
	if len(in.Providers) == 0 {
		return ClashConfigSubscription{}, ErrProvidersRequired
	}

	normalizedGroups, err := normalizeClashConfigProxyGroups(in.ProxyGroups)
	if err != nil {
		return ClashConfigSubscription{}, err
	}
	in.ProxyGroups = normalizedGroups

	return s.repo.CreateClashConfig(ctx, in)
}
```

- [ ] **Step 4: Re-run the reserved-row tests**

Run: `go test ./tests/integration -run 'TestCreateClashConfigSubscriptionRequiresReservedProxiesAtPositionZero|TestUpdateClashConfigSubscriptionAllowsEditingReservedProxiesMembersAndBinding' -v`
Expected: PASS

- [ ] **Step 5: Commit the reserved `Proxies` refactor**

```bash
git add internal/subscription/service.go internal/subscription/repository.go internal/subscription/model.go tests/integration/subscription_api_test.go
git commit -m "refactor: treat proxies as reserved ordered group"
```

## Task 3: Keep Output Rendering Aligned With Stored Order

**Files:**

- Modify: `internal/subscription/service.go`
- Modify: `tests/integration/subscription_output_test.go`

- [ ] **Step 1: Add a failing output test that proves rendered `proxy-groups` follow stored `position`**

```go
func TestClashConfigSubscriptionContentRendersProxyGroupsInStoredOrder(t *testing.T) {
	ts := newTestServerWithRefreshAndSubscriptions(t)
	defer ts.Close()

	providerID := seedProviderWithNodes(t, ts.URL, "alpha", "hk-01")
	groupID := createProxyGroup(t, ts.URL, `{"name":"Streaming","script":""}`)

	resp := postJSON(t, ts.URL+"/api/subscriptions/clash-configs", fmt.Sprintf(`{
		"name":"Daily",
		"providers":[%d],
		"proxy_groups":[
			{
				"name":"Proxies",
				"type":"select",
				"position":0,
				"proxies":[{"type":"DIRECT","value":"DIRECT"}],
				"bind_internal_proxy_group_id":%d
			},
			{
				"name":"Auto",
				"type":"url-test",
				"position":1,
				"url":"https://cp.cloudflare.com/generate_204",
				"interval":300,
				"proxies":[{"type":"internal","value":"%d"}],
				"bind_internal_proxy_group_id":%d
			},
			{
				"name":"Fallback",
				"type":"fallback",
				"position":2,
				"url":"https://cp.cloudflare.com/generate_204",
				"interval":300,
				"proxies":[{"type":"reference","value":"Auto"}],
				"bind_internal_proxy_group_id":%d
			}
		]
	}`, providerID, groupID, groupID, groupID, groupID))
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	contentResp, err := http.Get(ts.URL + "/api/subscriptions/clash-configs/1/content")
	require.NoError(t, err)
	defer contentResp.Body.Close()

	body := readBody(t, contentResp)
	assert.True(t, strings.Index(body, "name: Proxies") < strings.Index(body, "name: Auto"))
	assert.True(t, strings.Index(body, "name: Auto") < strings.Index(body, "name: Fallback"))
}
```

- [ ] **Step 2: Run the focused output test**

Run: `go test ./tests/integration -run TestClashConfigSubscriptionContentRendersProxyGroupsInStoredOrder -v`
Expected: FAIL if any content builder path still depends on prior `id` order or re-prepends `Proxies`.

- [ ] **Step 3: Remove any service-time reordering that conflicts with persisted `position`**

```go
func (s *Service) BuildClashConfigContent(ctx context.Context, id int64) (RenderedContent, error) {
	sub, err := s.repo.GetClashConfigByID(ctx, id)
	if err != nil {
		return RenderedContent{}, err
	}

	var renderedGroups []render.RenderedProxyGroup
	for _, pg := range sub.ProxyGroups {
		rp := render.RenderedProxyGroup{
			Name:     pg.Name,
			Type:     pg.Type,
			URL:      pg.URL,
			Interval: pg.Interval,
		}
		// existing member expansion logic stays here
		renderedGroups = append(renderedGroups, rp)
	}

	// render.RenderClashConfigSubscription should receive already-ordered groups
	// and must preserve that order.
	// ...
}
```

- [ ] **Step 4: Re-run the focused output test and the broader subscription-output suite**

Run: `go test ./tests/integration -run 'TestClashConfigSubscriptionContentRendersProxyGroupsInStoredOrder|TestClashConfigSubscriptionContentBuildsFromStoredComponents' -v`
Expected: PASS

- [ ] **Step 5: Commit the output-order alignment**

```bash
git add internal/subscription/service.go tests/integration/subscription_output_test.go
git commit -m "refactor: render clash config proxy groups in stored order"
```

## Task 4: Update the Clash Config Subscription UI

**Files:**

- Modify: `client/src/components/ClashConfigSubscriptionManager.tsx`

- [ ] **Step 1: Add a failing UX target as an inline checklist before changing the form**

```tsx
// Expected UI behavior:
// 1. "Add Clash Config Subscription" opens with a visible Proxies group already present.
// 2. Proxies appears first and has no delete control.
// 3. Proxies name and type are fixed, but its Rules Bound Group and Members remain editable.
// 4. Newly added non-system groups appear after Proxies.
```

- [ ] **Step 2: Run the frontend build to capture the current baseline**

Run: `cd client && npm run build`
Expected: PASS

- [ ] **Step 3: Refactor the form so `Proxies` is part of the editable list instead of being implied by the backend**

```tsx
const defaultProxiesGroup = {
  name: "Proxies",
  type: "select",
  position: 0,
  url: "",
  interval: 0,
  bind_internal_proxy_group_id: undefined,
  is_system: true,
  proxies: [{ type: "DIRECT", value: "DIRECT" }],
};
```

```tsx
const handleAdd = () => {
  setEditingSub(null);
  form.resetFields();
  form.setFieldsValue({
    name: "",
    providers: [],
    proxy_groups: [defaultProxiesGroup],
  });
  setModalVisible(true);
};
```

```tsx
const normalizedProxyGroups = (values.proxy_groups || []).map(
  (group: ProxyGroup, index: number) => ({
    ...group,
    position: index,
    is_system: index === 0 && group.name === "Proxies",
  }),
);
```

```tsx
<Form.Item
  name={[field.name, "name"]}
  label="Name"
  rules={[{ required: true }]}
>
  <Input placeholder="Media" disabled={isSystem} />
</Form.Item>

<Form.Item
  name={[field.name, "type"]}
  label="Type"
  rules={[{ required: true }]}
>
  <Select
    disabled={isSystem}
    options={[
      { label: "select", value: "select" },
      { label: "url-test", value: "url-test" },
      { label: "fallback", value: "fallback" },
    ]}
  />
</Form.Item>

<Form.Item
  name={[field.name, "bind_internal_proxy_group_id"]}
  label="Rules Bound Group"
  rules={[{ required: true, message: "Rules bound group is required" }]}
>
  <Select
    allowClear={!isSystem}
    placeholder="Select internal group"
    options={internalGroups.map((g) => ({
      label: g.name,
      value: g.id,
    }))}
  />
</Form.Item>
```

- [ ] **Step 4: Re-run the frontend build**

Run: `cd client && npm run build`
Expected: PASS

- [ ] **Step 5: Commit the UI refactor**

```bash
git add client/src/components/ClashConfigSubscriptionManager.tsx
git commit -m "refactor: surface reserved proxies group in ui"
```

## Task 5: End-to-End Verification

**Files:**

- Modify: `tests/integration/subscription_api_test.go`
- Modify: `tests/integration/subscription_output_test.go`

- [ ] **Step 1: Add regression coverage for migration backfill plus create-update-render flow**

```go
func TestSubscriptionMigrationBackfillsProxyGroupPositionsByLegacyIDOrder(t *testing.T) {
	// Seed legacy rows without position, run migration 004, and verify
	// each subscription receives contiguous positions matching prior id order.
}

func TestClashConfigSubscriptionUpdateReordersNonReservedProxyGroupsWithoutMovingProxies(t *testing.T) {
	// Create Proxies + two editable groups, update their order, and verify:
	// 1. Proxies stays at position 0
	// 2. non-reserved groups swap positions
	// 3. rendered yaml reflects the updated order
}
```

- [ ] **Step 2: Run the focused verification commands**

Run: `go test ./tests/integration -run 'TestSubscriptionMigrationBackfillsProxyGroupPositionsByLegacyIDOrder|TestClashConfigSubscriptionUpdateReordersNonReservedProxyGroupsWithoutMovingProxies' -v`
Expected: PASS

- [ ] **Step 3: Run the full verification commands**

Run: `go test ./...`
Expected: PASS

Run: `go vet ./...`
Expected: PASS

Run: `go build .`
Expected: PASS

Run: `cd client && npm run build`
Expected: PASS

- [ ] **Step 4: Commit the verification pass**

```bash
git add tests/integration/subscription_api_test.go tests/integration/subscription_output_test.go
git commit -m "test: verify phase4 proxy group refactor"
```

## Self-Review

- Spec coverage: this plan covers the new `position` column, migration `004` backfill behavior, reserved `Proxies` semantics, backend persistence, output rendering order, and the Clash config subscription UI changes.
- Placeholder scan: every task names exact files, target behavior, and concrete commands.
- Type consistency: the plan uses one persisted `position` field, one reserved `Proxies` row at position `0`, and one rule that `Proxies` is non-deletable but still editable for members and bound internal group.
