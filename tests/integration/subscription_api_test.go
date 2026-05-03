package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/EricWvi/subhub/internal/config"
	"github.com/EricWvi/subhub/internal/fetch"
	"github.com/EricWvi/subhub/internal/group"
	"github.com/EricWvi/subhub/internal/output"
	"github.com/EricWvi/subhub/internal/provider"
	"github.com/EricWvi/subhub/internal/refresh"
	"github.com/EricWvi/subhub/internal/rule"
	"github.com/EricWvi/subhub/internal/store"
	"github.com/EricWvi/subhub/internal/subscription"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServerWithSubscriptions(t *testing.T) *httptest.Server {
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

	fetcher := fetch.NewClient(cfg.UpstreamRequestTimeout)
	refreshSvc := refresh.NewService(providerRepo, fetcher)
	providerHandler.SetRefresher(refreshSvc.RefreshProvider)

	groupRepo := group.NewRepository(db)
	groupSvc := group.NewService(groupRepo)
	groupHandler := group.NewHandler(groupSvc)

	ruleRepo := rule.NewRepository(db)
	ruleSvc := rule.NewService(ruleRepo)
	ruleHandler := rule.NewHandler(ruleSvc)

	outputHandler := output.NewHandler(providerRepo, ruleRepo, filepath.Join("..", "fixtures", "template.yaml"))

	subscriptionRepo := subscription.NewRepository(db)
	subscriptionSvc := subscription.NewService(subscriptionRepo, providerRepo, groupSvc, ruleRepo, filepath.Join("..", "fixtures", "client_sub.yaml"))
	subscriptionHandler := subscription.NewHandler(subscriptionSvc)

	providerSvc.SetSubscriptionReferenceChecker(subscriptionSvc.ProviderReferencedByAnySubscription)

	apiMux := http.NewServeMux()
	providerHandler.RegisterRoutes(apiMux)
	groupHandler.RegisterRoutes(apiMux)
	ruleHandler.RegisterRoutes(apiMux)
	outputHandler.RegisterRoutes(apiMux)
	subscriptionHandler.RegisterRoutes(apiMux)

	mux := http.NewServeMux()
	mux.Handle("/api/", http.StripPrefix("/api", apiMux))
	return httptest.NewServer(mux)
}

func TestListClashConfigSubscriptionsStartsEmpty(t *testing.T) {
	ts := newTestServerWithSubscriptions(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/subscriptions/clash-configs")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.JSONEq(t, `{"subscriptions":[]}`, readBody(t, resp))
}

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
				"name":"Proxies",
				"type":"select",
				"position":0,
				"proxies":[{"type":"DIRECT","value":"DIRECT"}],
				"bind_internal_proxy_group_id":%d
			},
			{
				"name":"Media",
				"type":"url-test",
				"position":9,
				"url":"https://cp.cloudflare.com/generate_204",
				"interval":300,
				"proxies":[{"type":"internal","value":"%d"},{"type":"DIRECT","value":"DIRECT"}],
				"bind_internal_proxy_group_id":%d
			}
		]
	}`, p1, p2, g1, g1, g2))
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
				"name":"Proxies",
				"type":"select",
				"position":0,
				"proxies":[{"type":"internal","value":"%d"},{"type":"DIRECT","value":"DIRECT"}],
				"bind_internal_proxy_group_id":%d
			},
			{
				"name":"Media",
				"type":"fallback",
				"position":4,
				"url":"https://cp.cloudflare.com/generate_204",
				"interval":300,
				"proxies":[{"type":"internal","value":"%d"},{"type":"REJECT","value":"REJECT"}],
				"bind_internal_proxy_group_id":%d
			}
		]
	}`, p2, g2, g2, g1, g1))
	defer updateResp.Body.Close()
	assert.Equal(t, http.StatusOK, updateResp.StatusCode)

	deleteResp := deleteRequest(t, ts.URL+"/api/subscriptions/clash-configs/1")
	defer deleteResp.Body.Close()
	assert.Equal(t, http.StatusNoContent, deleteResp.StatusCode)
}

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
	getResp, err := http.Get(ts.URL + "/api/subscriptions/clash-configs/1")
	require.NoError(t, err)
	defer getResp.Body.Close()

	require.Equal(t, http.StatusOK, getResp.StatusCode)
	body := readBody(t, getResp)
	assert.Regexp(t, `"proxy_groups":\[\{"id":[0-9]+,"name":"Proxies".*"position":0`, body)
	assert.Contains(t, body, `"name":"Fallback","type":"fallback","position":1`)
	assert.Contains(t, body, `"name":"Auto","type":"url-test","position":2`)
	assert.True(t, strings.Index(body, `"name":"Proxies"`) < strings.Index(body, `"name":"Fallback"`))
	assert.True(t, strings.Index(body, `"name":"Fallback"`) < strings.Index(body, `"name":"Auto"`))
}

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
	body := readBody(t, resp)
	assert.Contains(t, body, "Proxies")
	assert.NotContains(t, body, "at least one provider is required")
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
				"proxies":[{"type":"internal","value":"%d"},{"type":"DIRECT","value":"DIRECT"}],
				"bind_internal_proxy_group_id":%d
			}
		]
	}`, providerID, groupB, groupB))
	defer updateResp.Body.Close()
	require.Equal(t, http.StatusOK, updateResp.StatusCode)

	body := readBody(t, updateResp)
	assert.Contains(t, body, `"name":"Proxies"`)
	assert.Contains(t, body, `"position":0`)
	assert.Contains(t, body, fmt.Sprintf(`"bind_internal_proxy_group_id":%d`, groupB))
	assert.Contains(t, body, fmt.Sprintf(`"type":"internal","value":"%d"`, groupB))
}

func TestCreateClashConfigSubscriptionAcceptsReservedProxiesFirstWithNonZeroSubmittedPosition(t *testing.T) {
	ts := newTestServerWithSubscriptions(t)
	defer ts.Close()

	providerID := createProvider(t, ts.URL, `{"name":"alpha","url":"https://example.com/a"}`)
	groupID := createProxyGroup(t, ts.URL, `{"name":"Streaming","script":""}`)

	resp := postJSON(t, ts.URL+"/api/subscriptions/clash-configs", fmt.Sprintf(`{
		"name":"Daily",
		"providers":[%d],
		"proxy_groups":[
			{
				"name":"Proxies",
				"type":"select",
				"position":9,
				"proxies":[{"type":"DIRECT","value":"DIRECT"}],
				"bind_internal_proxy_group_id":%d
			}
		]
	}`, providerID, groupID))
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode)
	body := readBody(t, resp)
	assert.Contains(t, body, `"name":"Proxies"`)
	assert.Contains(t, body, `"position":0`)
	assert.NotContains(t, body, `"position":9`)
}

func TestCreateClashConfigSubscriptionRejectsReservedProxiesWhenPresentButNotFirst(t *testing.T) {
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
				"type":"fallback",
				"position":0,
				"url":"https://cp.cloudflare.com/generate_204",
				"interval":300,
				"proxies":[{"type":"DIRECT","value":"DIRECT"}],
				"bind_internal_proxy_group_id":%d
			},
			{
				"name":"Proxies",
				"type":"select",
				"position":1,
				"proxies":[{"type":"DIRECT","value":"DIRECT"}],
				"bind_internal_proxy_group_id":%d
			}
		]
	}`, providerID, groupID, groupID))
	defer resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, readBody(t, resp), "Proxies")
}

func TestUpdateClashConfigSubscriptionRejectsOmittingReservedProxies(t *testing.T) {
	ts := newTestServerWithSubscriptions(t)
	defer ts.Close()

	providerID := createProvider(t, ts.URL, `{"name":"alpha","url":"https://example.com/a"}`)
	groupID := createProxyGroup(t, ts.URL, `{"name":"A","script":""}`)

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
	}`, providerID, groupID))
	defer createResp.Body.Close()
	require.Equal(t, http.StatusCreated, createResp.StatusCode)

	updateResp := putJSON(t, ts.URL+"/api/subscriptions/clash-configs/1", fmt.Sprintf(`{
		"name":"Daily",
		"providers":[%d],
		"proxy_groups":[]
	}`, providerID))
	defer updateResp.Body.Close()

	require.Equal(t, http.StatusBadRequest, updateResp.StatusCode)
	assert.Contains(t, readBody(t, updateResp), "Proxies")
}

func TestUpdateClashConfigSubscriptionRejectsReservedProxiesRenameOrNonSelectType(t *testing.T) {
	ts := newTestServerWithSubscriptions(t)
	defer ts.Close()

	providerID := createProvider(t, ts.URL, `{"name":"alpha","url":"https://example.com/a"}`)
	groupID := createProxyGroup(t, ts.URL, `{"name":"A","script":""}`)

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
	}`, providerID, groupID))
	defer createResp.Body.Close()
	require.Equal(t, http.StatusCreated, createResp.StatusCode)

	renameResp := putJSON(t, ts.URL+"/api/subscriptions/clash-configs/1", fmt.Sprintf(`{
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
	defer renameResp.Body.Close()

	require.Equal(t, http.StatusBadRequest, renameResp.StatusCode)
	assert.Contains(t, readBody(t, renameResp), "Proxies")

	typeResp := putJSON(t, ts.URL+"/api/subscriptions/clash-configs/1", fmt.Sprintf(`{
		"name":"Daily",
		"providers":[%d],
		"proxy_groups":[
			{
				"name":"Proxies",
				"type":"fallback",
				"position":0,
				"url":"https://cp.cloudflare.com/generate_204",
				"interval":300,
				"proxies":[{"type":"DIRECT","value":"DIRECT"}],
				"bind_internal_proxy_group_id":%d
			}
		]
	}`, providerID, groupID))
	defer typeResp.Body.Close()

	require.Equal(t, http.StatusBadRequest, typeResp.StatusCode)
	assert.Contains(t, readBody(t, typeResp), "Proxies")
}

func TestUpdateClashConfigSubscriptionReordersNonReservedProxyGroupsWithoutMovingProxies(t *testing.T) {
	ts := newTestServerWithSubscriptions(t)
	defer ts.Close()

	providerID := createProvider(t, ts.URL, `{"name":"alpha","url":"https://example.com/a"}`)
	proxiesGroupID := createProxyGroup(t, ts.URL, `{"name":"Default","script":""}`)
	autoGroupID := createProxyGroup(t, ts.URL, `{"name":"AutoNodes","script":""}`)
	fallbackGroupID := createProxyGroup(t, ts.URL, `{"name":"FallbackNodes","script":""}`)

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
			},
			{
				"name":"Auto",
				"type":"select",
				"position":1,
				"proxies":[{"type":"internal","value":"%d"},{"type":"DIRECT","value":"DIRECT"}],
				"bind_internal_proxy_group_id":%d
			},
			{
				"name":"Fallback",
				"type":"select",
				"position":2,
				"proxies":[{"type":"internal","value":"%d"},{"type":"DIRECT","value":"DIRECT"}],
				"bind_internal_proxy_group_id":%d
			}
		]
	}`, providerID, proxiesGroupID, autoGroupID, autoGroupID, fallbackGroupID, fallbackGroupID))
	defer createResp.Body.Close()
	require.Equal(t, http.StatusCreated, createResp.StatusCode)

	updateResp := putJSON(t, ts.URL+"/api/subscriptions/clash-configs/1", fmt.Sprintf(`{
		"name":"Daily",
		"providers":[%d],
		"proxy_groups":[
			{
				"name":"Proxies",
				"type":"select",
				"position":99,
				"proxies":[{"type":"DIRECT","value":"DIRECT"}],
				"bind_internal_proxy_group_id":%d
			},
			{
				"name":"Fallback",
				"type":"select",
				"position":42,
				"proxies":[{"type":"internal","value":"%d"},{"type":"DIRECT","value":"DIRECT"}],
				"bind_internal_proxy_group_id":%d
			},
			{
				"name":"Auto",
				"type":"select",
				"position":7,
				"proxies":[{"type":"internal","value":"%d"},{"type":"DIRECT","value":"DIRECT"}],
				"bind_internal_proxy_group_id":%d
			}
		]
	}`, providerID, proxiesGroupID, fallbackGroupID, fallbackGroupID, autoGroupID, autoGroupID))
	defer updateResp.Body.Close()
	require.Equal(t, http.StatusOK, updateResp.StatusCode)

	body := readBody(t, updateResp)
	assert.True(t, strings.Index(body, `"name":"Proxies"`) < strings.Index(body, `"name":"Fallback"`))
	assert.True(t, strings.Index(body, `"name":"Fallback"`) < strings.Index(body, `"name":"Auto"`))
	assert.Contains(t, body, `"name":"Proxies","type":"select","position":0`)
	assert.Contains(t, body, `"name":"Fallback","type":"select","position":1`)
	assert.Contains(t, body, `"name":"Auto","type":"select","position":2`)
	assert.NotContains(t, body, `"position":99`)
	assert.NotContains(t, body, `"position":42`)
	assert.NotContains(t, body, `"position":7`)
}
