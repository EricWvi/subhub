package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
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
