package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/EricWvi/subhub/internal/config"
	"github.com/EricWvi/subhub/internal/fetch"
	"github.com/EricWvi/subhub/internal/group"
	"github.com/EricWvi/subhub/internal/provider"
	"github.com/EricWvi/subhub/internal/refresh"
	"github.com/EricWvi/subhub/internal/rule"
	"github.com/EricWvi/subhub/internal/store"
	"github.com/EricWvi/subhub/internal/subscription"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServerWithSubscriptionOutput(t *testing.T) (*httptest.Server, *provider.Repository) {
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

	subscriptionRepo := subscription.NewRepository(db)
	subscriptionSvc := subscription.NewService(subscriptionRepo, providerRepo, groupSvc, ruleRepo, filepath.Join("..", "fixtures", "client_sub.yaml"))
	subscriptionHandler := subscription.NewHandler(subscriptionSvc)

	providerSvc.SetSubscriptionReferenceChecker(subscriptionSvc.ProviderReferencedByAnySubscription)

	apiMux := http.NewServeMux()
	providerHandler.RegisterRoutes(apiMux)
	groupHandler.RegisterRoutes(apiMux)
	ruleHandler.RegisterRoutes(apiMux)
	subscriptionHandler.RegisterRoutes(apiMux)

	mux := http.NewServeMux()
	mux.Handle("/api/", http.StripPrefix("/api", apiMux))
	return httptest.NewServer(mux), providerRepo
}

func TestClashConfigContentBuildsFromStoredComponents(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "fixtures", "provider_plain.yaml"))
	require.NoError(t, err)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/yaml")
		w.Header().Set("Subscription-Userinfo", "upload=1024; download=2048; total=4096; expire=1893456000")
		w.Write(fixture)
	}))
	defer upstream.Close()

	ts, _ := newTestServerWithSubscriptionOutput(t)
	defer ts.Close()

	providerID := createProvider(t, ts.URL, fmt.Sprintf(`{"name":"alpha","url":"%s"}`, upstream.URL))
	refreshResp := refreshProvider(t, ts.URL, providerID)
	defer refreshResp.Body.Close()
	require.Equal(t, http.StatusNoContent, refreshResp.StatusCode)

	proxiesGroupID := createProxyGroup(t, ts.URL, `{"name":"DefaultNodes","script":""}`)
	internalGroupID := createProxyGroup(t, ts.URL, `{"name":"AllNodes","script":""}`)

	createResp := postJSON(t, ts.URL+"/api/subscriptions/clash-configs", fmt.Sprintf(`{
		"name":"Test Sub",
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
				"name":"MyProxies",
				"type":"select",
				"position":1,
				"proxies":[{"type":"internal","value":"%d"},{"type":"DIRECT","value":"DIRECT"}],
				"bind_internal_proxy_group_id":%d
			}
		]
	}`, providerID, proxiesGroupID, internalGroupID, internalGroupID))
	defer createResp.Body.Close()
	require.Equal(t, http.StatusCreated, createResp.StatusCode)

	contentResp, err := http.Get(ts.URL + "/api/subscriptions/clash-configs/1/content")
	require.NoError(t, err)
	defer contentResp.Body.Close()
	assert.Equal(t, http.StatusOK, contentResp.StatusCode)
	assert.Equal(t, "application/yaml", contentResp.Header.Get("Content-Type"))
	assert.Equal(t, "upload=0; download=3072; total=4096; expire=1893456000", contentResp.Header.Get("Subscription-Userinfo"))

	body := readBody(t, contentResp)
	assert.Contains(t, body, "name: vmess-hk-01")
	assert.Contains(t, body, "name: ss-jp-01")
	assert.Contains(t, body, "name: Proxies")
	assert.Contains(t, body, "name: MyProxies")
	assert.Contains(t, body, "name: Final")
	assert.Contains(t, body, "rules:")
}

func TestClashConfigSubscriptionContentRendersProxyGroupsInStoredOrder(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "fixtures", "provider_plain.yaml"))
	require.NoError(t, err)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/yaml")
		w.Header().Set("Subscription-Userinfo", "upload=0; download=100; total=1000; expire=1893456000")
		w.Write(fixture)
	}))
	defer upstream.Close()

	ts, _ := newTestServerWithSubscriptionOutput(t)
	defer ts.Close()

	providerID := createProvider(t, ts.URL, fmt.Sprintf(`{"name":"alpha","url":"%s"}`, upstream.URL))
	refreshResp := refreshProvider(t, ts.URL, providerID)
	defer refreshResp.Body.Close()
	require.Equal(t, http.StatusNoContent, refreshResp.StatusCode)

	proxiesGroupID := createProxyGroup(t, ts.URL, `{"name":"Streaming","script":""}`)
	autoGroupID := createProxyGroup(t, ts.URL, `{"name":"AutoNodes","script":""}`)
	fallbackGroupID := createProxyGroup(t, ts.URL, `{"name":"FallbackNodes","script":""}`)

	createResp := postJSON(t, ts.URL+"/api/subscriptions/clash-configs", fmt.Sprintf(`{
		"name":"Stored Order",
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
	}`, providerID, proxiesGroupID, autoGroupID, autoGroupID, fallbackGroupID))
	defer createResp.Body.Close()
	require.Equal(t, http.StatusCreated, createResp.StatusCode)

	contentResp, err := http.Get(ts.URL + "/api/subscriptions/clash-configs/1/content")
	require.NoError(t, err)
	defer contentResp.Body.Close()
	require.Equal(t, http.StatusOK, contentResp.StatusCode)

	body := readBody(t, contentResp)
	proxiesIndex := strings.Index(body, "name: Proxies")
	autoIndex := strings.Index(body, "name: Auto")
	fallbackIndex := strings.Index(body, "name: Fallback")
	require.NotEqual(t, -1, proxiesIndex)
	require.NotEqual(t, -1, autoIndex)
	require.NotEqual(t, -1, fallbackIndex)
	assert.Less(t, proxiesIndex, autoIndex)
	assert.Less(t, autoIndex, fallbackIndex)
}

func TestClashConfigContentExcludesNodesWithoutConcreteNonReferenceMembers(t *testing.T) {
	fixture := []byte("proxies:\n  - {name: HK, type: vmess, server: hk.example.com, port: 443}\n  - {name: JP, type: vmess, server: jp.example.com, port: 443}\n")

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/yaml")
		w.Header().Set("Subscription-Userinfo", "upload=0; download=100; total=1000; expire=1893456000")
		w.Write(fixture)
	}))
	defer upstream.Close()

	ts, _ := newTestServerWithSubscriptionOutput(t)
	defer ts.Close()

	providerID := createProvider(t, ts.URL, fmt.Sprintf(`{"name":"alpha","url":"%s"}`, upstream.URL))
	refreshResp := refreshProvider(t, ts.URL, providerID)
	defer refreshResp.Body.Close()
	require.Equal(t, http.StatusNoContent, refreshResp.StatusCode)

	proxiesGroupID := createProxyGroup(t, ts.URL, `{"name":"DefaultNodes","script":""}`)

	createResp := postJSON(t, ts.URL+"/api/subscriptions/clash-configs", fmt.Sprintf(`{
		"name":"Keep Nodes",
		"providers":[%d],
		"proxy_groups":[
			{
				"name":"Proxies",
				"type":"select",
				"position":0,
				"proxies":[{"type":"reference","value":"HK"},{"type":"DIRECT","value":"DIRECT"}],
				"bind_internal_proxy_group_id":%d
			}
		]
	}`, providerID, proxiesGroupID))
	defer createResp.Body.Close()
	require.Equal(t, http.StatusCreated, createResp.StatusCode)

	updateResp := putJSON(t, ts.URL+"/api/subscriptions/clash-configs/1", fmt.Sprintf(`{
		"name":"Keep Nodes",
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
	}`, providerID, proxiesGroupID))
	defer updateResp.Body.Close()
	require.Equal(t, http.StatusOK, updateResp.StatusCode)

	contentResp, err := http.Get(ts.URL + "/api/subscriptions/clash-configs/1/content")
	require.NoError(t, err)
	defer contentResp.Body.Close()
	require.Equal(t, http.StatusOK, contentResp.StatusCode)

	body := readBody(t, contentResp)
	assert.NotContains(t, body, "name: HK")
	assert.NotContains(t, body, "name: JP")
	assert.Contains(t, body, "- DIRECT")
}

func TestClashConfigContentReturns404ForMissing(t *testing.T) {
	ts, _ := newTestServerWithSubscriptionOutput(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/subscriptions/clash-configs/999/content")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestClashConfigContentRemapsRulesToOutputGroups(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "fixtures", "provider_plain.yaml"))
	require.NoError(t, err)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/yaml")
		w.Header().Set("Subscription-Userinfo", "upload=0; download=100; total=1000; expire=1893456000")
		w.Write(fixture)
	}))
	defer upstream.Close()

	ts, _ := newTestServerWithSubscriptionOutput(t)
	defer ts.Close()

	providerID := createProvider(t, ts.URL, fmt.Sprintf(`{"name":"alpha","url":"%s"}`, upstream.URL))
	refreshResp := refreshProvider(t, ts.URL, providerID)
	defer refreshResp.Body.Close()
	require.Equal(t, http.StatusNoContent, refreshResp.StatusCode)

	proxiesGroupID := createProxyGroup(t, ts.URL, `{"name":"DefaultStreaming","script":""}`)
	internalGroupID := createProxyGroup(t, ts.URL, `{"name":"Streaming","script":""}`)

	postJSON(t, ts.URL+"/api/rules", fmt.Sprintf(`{"rule_type":"DOMAIN-SUFFIX","pattern":"netflix.com","proxy_group":"Streaming"}`)).Body.Close()
	postJSON(t, ts.URL+"/api/rules", `{"rule_type":"DOMAIN-KEYWORD","pattern":"openai","proxy_group":"DIRECT"}`).Body.Close()

	createResp := postJSON(t, ts.URL+"/api/subscriptions/clash-configs", fmt.Sprintf(`{
		"name":"Remap Test",
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
				"name":"Media",
				"type":"select",
				"position":1,
				"proxies":[{"type":"internal","value":"%d"},{"type":"DIRECT","value":"DIRECT"}],
				"bind_internal_proxy_group_id":%d
			}
		]
	}`, providerID, proxiesGroupID, internalGroupID, internalGroupID))
	defer createResp.Body.Close()
	require.Equal(t, http.StatusCreated, createResp.StatusCode)

	contentResp, err := http.Get(ts.URL + "/api/subscriptions/clash-configs/1/content")
	require.NoError(t, err)
	defer contentResp.Body.Close()
	require.Equal(t, http.StatusOK, contentResp.StatusCode)

	body := readBody(t, contentResp)
	assert.Contains(t, body, "DOMAIN-SUFFIX,netflix.com,Media")
	assert.Contains(t, body, "DOMAIN-KEYWORD,openai,DIRECT")
}

func TestClashConfigContentSkipsRulesThatDoNotMapToOutputGroups(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "fixtures", "provider_plain.yaml"))
	require.NoError(t, err)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/yaml")
		w.Header().Set("Subscription-Userinfo", "upload=0; download=100; total=1000; expire=1893456000")
		w.Write(fixture)
	}))
	defer upstream.Close()

	ts, _ := newTestServerWithSubscriptionOutput(t)
	defer ts.Close()

	providerID := createProvider(t, ts.URL, fmt.Sprintf(`{"name":"alpha","url":"%s"}`, upstream.URL))
	refreshResp := refreshProvider(t, ts.URL, providerID)
	defer refreshResp.Body.Close()
	require.Equal(t, http.StatusNoContent, refreshResp.StatusCode)

	proxiesGroupID := createProxyGroup(t, ts.URL, `{"name":"DefaultStreaming","script":""}`)
	internalGroupID := createProxyGroup(t, ts.URL, `{"name":"Streaming","script":""}`)
	createProxyGroup(t, ts.URL, `{"name":"Blocked","script":""}`)

	postJSON(t, ts.URL+"/api/rules", fmt.Sprintf(`{"rule_type":"DOMAIN-SUFFIX","pattern":"netflix.com","proxy_group":"Streaming"}`)).Body.Close()
	postJSON(t, ts.URL+"/api/rules", fmt.Sprintf(`{"rule_type":"DOMAIN-SUFFIX","pattern":"blocked.com","proxy_group":"Blocked"}`)).Body.Close()

	createResp := postJSON(t, ts.URL+"/api/subscriptions/clash-configs", fmt.Sprintf(`{
		"name":"Remap Test",
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
				"name":"Media",
				"type":"select",
				"position":1,
				"proxies":[{"type":"internal","value":"%d"},{"type":"DIRECT","value":"DIRECT"}],
				"bind_internal_proxy_group_id":%d
			}
		]
	}`, providerID, proxiesGroupID, internalGroupID, internalGroupID))
	defer createResp.Body.Close()
	require.Equal(t, http.StatusCreated, createResp.StatusCode)

	contentResp, err := http.Get(ts.URL + "/api/subscriptions/clash-configs/1/content")
	require.NoError(t, err)
	defer contentResp.Body.Close()
	require.Equal(t, http.StatusOK, contentResp.StatusCode)

	body := readBody(t, contentResp)
	assert.Contains(t, body, "DOMAIN-SUFFIX,netflix.com,Media")
	assert.NotContains(t, body, "blocked.com")
}

func TestClashConfigContentExcludesInvalidProviders(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "fixtures", "provider_plain.yaml"))
	require.NoError(t, err)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/yaml")
		w.Write(fixture)
	}))
	defer upstream.Close()

	ts, _ := newTestServerWithSubscriptionOutput(t)
	defer ts.Close()

	providerID := createProvider(t, ts.URL, fmt.Sprintf(`{"name":"alpha","url":"%s"}`, upstream.URL))
	refreshResp := refreshProvider(t, ts.URL, providerID)
	defer refreshResp.Body.Close()
	require.Equal(t, http.StatusNoContent, refreshResp.StatusCode)

	proxiesGroupID := createProxyGroup(t, ts.URL, `{"name":"DefaultNodes","script":""}`)
	internalGroupID := createProxyGroup(t, ts.URL, `{"name":"AllNodes","script":""}`)

	createResp := postJSON(t, ts.URL+"/api/subscriptions/clash-configs", fmt.Sprintf(`{
		"name":"No Valid",
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
				"name":"MyProxies",
				"type":"select",
				"position":1,
				"proxies":[{"type":"internal","value":"%d"},{"type":"DIRECT","value":"DIRECT"}],
				"bind_internal_proxy_group_id":%d
			}
		]
	}`, providerID, proxiesGroupID, internalGroupID, internalGroupID))
	defer createResp.Body.Close()
	require.Equal(t, http.StatusCreated, createResp.StatusCode)

	contentResp, err := http.Get(ts.URL + "/api/subscriptions/clash-configs/1/content")
	require.NoError(t, err)
	defer contentResp.Body.Close()
	assert.Equal(t, http.StatusOK, contentResp.StatusCode)
	assert.Empty(t, contentResp.Header.Get("Subscription-Userinfo"))

	body := readBody(t, contentResp)
	assert.NotContains(t, body, "vmess-hk-01")
}

func TestClashConfigContentIncludesTemplateRules(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "fixtures", "provider_plain.yaml"))
	require.NoError(t, err)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/yaml")
		w.Header().Set("Subscription-Userinfo", "upload=0; download=100; total=1000; expire=1893456000")
		w.Write(fixture)
	}))
	defer upstream.Close()

	ts, _ := newTestServerWithSubscriptionOutput(t)
	defer ts.Close()

	providerID := createProvider(t, ts.URL, fmt.Sprintf(`{"name":"alpha","url":"%s"}`, upstream.URL))
	refreshResp := refreshProvider(t, ts.URL, providerID)
	defer refreshResp.Body.Close()
	require.Equal(t, http.StatusNoContent, refreshResp.StatusCode)

	proxiesGroupID := createProxyGroup(t, ts.URL, `{"name":"DefaultNodes","script":""}`)
	internalGroupID := createProxyGroup(t, ts.URL, `{"name":"AllNodes","script":""}`)

	createResp := postJSON(t, ts.URL+"/api/subscriptions/clash-configs", fmt.Sprintf(`{
		"name":"Template Rules",
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
				"name":"MyProxies",
				"type":"select",
				"position":1,
				"proxies":[{"type":"internal","value":"%d"},{"type":"DIRECT","value":"DIRECT"}],
				"bind_internal_proxy_group_id":%d
			}
		]
	}`, providerID, proxiesGroupID, internalGroupID, internalGroupID))
	defer createResp.Body.Close()
	require.Equal(t, http.StatusCreated, createResp.StatusCode)

	contentResp, err := http.Get(ts.URL + "/api/subscriptions/clash-configs/1/content")
	require.NoError(t, err)
	defer contentResp.Body.Close()
	require.Equal(t, http.StatusOK, contentResp.StatusCode)

	body := readBody(t, contentResp)
	assert.True(t, strings.Contains(body, "GEOIP,CN,DIRECT"), "should contain template GEOIP rule")
	assert.True(t, strings.Contains(body, "MATCH,Final"), "should contain template MATCH rule")
}

func TestProxyProviderSubscriptionContentExportsYamlProxies(t *testing.T) {
	ts := newTestServerWithSubscriptions(t)
	defer ts.Close()

	fixture := []byte("proxies:\n  - {name: hk-01, type: vmess, server: hk.example.com, port: 443}\n  - {name: hk-02, type: vmess, server: hk2.example.com, port: 443}\n")
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/yaml")
		w.Header().Set("Subscription-Userinfo", "upload=0; download=100; total=1000; expire=1893456000")
		w.Write(fixture)
	}))
	defer upstream.Close()

	providerID := createProvider(t, ts.URL, fmt.Sprintf(`{"name":"alpha","url":"%s"}`, upstream.URL))
	refreshResp := refreshProvider(t, ts.URL, providerID)
	defer refreshResp.Body.Close()
	require.Equal(t, http.StatusNoContent, refreshResp.StatusCode)

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
}

func TestRuleProviderSubscriptionContentExportsYamlPayload(t *testing.T) {
	ts := newTestServerWithSubscriptions(t)
	defer ts.Close()

	providerID := createProvider(t, ts.URL, `{"name":"alpha","url":"https://example.com/a"}`)
	groupID := createProxyGroup(t, ts.URL, `{"name":"Rules","script":""}`)
	postJSON(t, ts.URL+"/api/rules", `{"rule_type":"DOMAIN-SUFFIX","pattern":"google.com","proxy_group":"Rules"}`).Body.Close()

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
	body := readBody(t, contentResp)
	assert.Contains(t, body, "payload:")
	assert.Contains(t, body, "DOMAIN-SUFFIX,google.com")
}

func TestClashConfigSubscriptionContentRendersUpdatedProxyGroupOrder(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "fixtures", "provider_plain.yaml"))
	require.NoError(t, err)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/yaml")
		w.Header().Set("Subscription-Userinfo", "upload=0; download=100; total=1000; expire=1893456000")
		w.Write(fixture)
	}))
	defer upstream.Close()

	ts, _ := newTestServerWithSubscriptionOutput(t)
	defer ts.Close()

	providerID := createProvider(t, ts.URL, fmt.Sprintf(`{"name":"alpha","url":"%s"}`, upstream.URL))
	refreshResp := refreshProvider(t, ts.URL, providerID)
	defer refreshResp.Body.Close()
	require.Equal(t, http.StatusNoContent, refreshResp.StatusCode)

	proxiesGroupID := createProxyGroup(t, ts.URL, `{"name":"DefaultNodes","script":""}`)
	autoGroupID := createProxyGroup(t, ts.URL, `{"name":"AutoNodes","script":""}`)
	fallbackGroupID := createProxyGroup(t, ts.URL, `{"name":"FallbackNodes","script":""}`)

	createResp := postJSON(t, ts.URL+"/api/subscriptions/clash-configs", fmt.Sprintf(`{
		"name":"Render Update",
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
		"name":"Render Update",
		"providers":[%d],
		"proxy_groups":[
			{
				"name":"Proxies",
				"type":"select",
				"position":50,
				"proxies":[{"type":"DIRECT","value":"DIRECT"}],
				"bind_internal_proxy_group_id":%d
			},
			{
				"name":"Fallback",
				"type":"select",
				"position":20,
				"proxies":[{"type":"internal","value":"%d"},{"type":"DIRECT","value":"DIRECT"}],
				"bind_internal_proxy_group_id":%d
			},
			{
				"name":"Auto",
				"type":"select",
				"position":10,
				"proxies":[{"type":"internal","value":"%d"},{"type":"DIRECT","value":"DIRECT"}],
				"bind_internal_proxy_group_id":%d
			}
		]
	}`, providerID, proxiesGroupID, fallbackGroupID, fallbackGroupID, autoGroupID, autoGroupID))
	defer updateResp.Body.Close()
	require.Equal(t, http.StatusOK, updateResp.StatusCode)

	contentResp, err := http.Get(ts.URL + "/api/subscriptions/clash-configs/1/content")
	require.NoError(t, err)
	defer contentResp.Body.Close()
	require.Equal(t, http.StatusOK, contentResp.StatusCode)

	body := readBody(t, contentResp)
	proxiesIndex := strings.Index(body, "name: Proxies")
	fallbackIndex := strings.Index(body, "name: Fallback")
	autoIndex := strings.Index(body, "name: Auto")
	require.NotEqual(t, -1, proxiesIndex)
	require.NotEqual(t, -1, fallbackIndex)
	require.NotEqual(t, -1, autoIndex)
	assert.Less(t, proxiesIndex, fallbackIndex)
	assert.Less(t, fallbackIndex, autoIndex)
}
