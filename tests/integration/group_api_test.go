package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/EricWvi/subhub/internal/config"
	"github.com/EricWvi/subhub/internal/fetch"
	"github.com/EricWvi/subhub/internal/group"
	"github.com/EricWvi/subhub/internal/output"
	"github.com/EricWvi/subhub/internal/provider"
	"github.com/EricWvi/subhub/internal/refresh"
	"github.com/EricWvi/subhub/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServerWithGroups(t *testing.T) *httptest.Server {
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

	repo := provider.NewRepository(db)
	svc := provider.NewService(repo)
	handler := provider.NewHandler(svc)

	groupRepo := group.NewRepository(db)
	groupSvc := group.NewService(groupRepo)
	groupHandler := group.NewHandler(groupSvc)

	apiMux := http.NewServeMux()
	handler.RegisterRoutes(apiMux)
	groupHandler.RegisterRoutes(apiMux)

	mux := http.NewServeMux()
	mux.Handle("/api/", http.StripPrefix("/api", apiMux))
	return httptest.NewServer(mux)
}

func newTestServerWithRefreshAndGroups(t *testing.T) (*httptest.Server, *provider.Repository) {
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

	repo := provider.NewRepository(db)
	svc := provider.NewService(repo)
	handler := provider.NewHandler(svc)

	fetcher := fetch.NewClient(cfg.UpstreamRequestTimeout)
	refreshSvc := refresh.NewService(repo, fetcher)
	handler.SetRefresher(refreshSvc.RefreshProvider)

	templatePath := filepath.Join("..", "fixtures", "template.yaml")
	outputHandler := output.NewHandler(repo, templatePath)

	groupRepo := group.NewRepository(db)
	groupSvc := group.NewService(groupRepo)
	groupHandler := group.NewHandler(groupSvc)

	apiMux := http.NewServeMux()
	handler.RegisterRoutes(apiMux)
	outputHandler.RegisterRoutes(apiMux)
	groupHandler.RegisterRoutes(apiMux)

	mux := http.NewServeMux()
	mux.Handle("/api/", http.StripPrefix("/api", apiMux))
	return httptest.NewServer(mux), repo
}

func TestListProxyGroupsStartsEmpty(t *testing.T) {
	ts := newTestServerWithGroups(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/proxy-groups")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.JSONEq(t, `{"groups":[]}`, readBody(t, resp))
}

func TestCreateGetUpdateDeleteProxyGroup(t *testing.T) {
	ts := newTestServerWithGroups(t)
	defer ts.Close()

	createResp := postJSON(t, ts.URL+"/api/proxy-groups", `{"name":"OpenAI","script":"function (proxyNodes) { return proxyNodes.map(function (node) { return node.id }) }"}`)
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

	getResp, err := http.Get(fmt.Sprintf("%s/api/proxy-groups/%d", ts.URL, created.Group.ID))
	require.NoError(t, err)
	defer getResp.Body.Close()
	assert.Equal(t, http.StatusOK, getResp.StatusCode)
	assert.Contains(t, readBody(t, getResp), `"name":"OpenAI"`)

	updateResp := putJSON(t, fmt.Sprintf("%s/api/proxy-groups/%d", ts.URL, created.Group.ID), `{"name":"Streaming","script":""}`)
	defer updateResp.Body.Close()
	assert.Equal(t, http.StatusOK, updateResp.StatusCode)
	assert.Contains(t, readBody(t, updateResp), `"script":""`)

	deleteResp := deleteRequest(t, fmt.Sprintf("%s/api/proxy-groups/%d", ts.URL, created.Group.ID))
	defer deleteResp.Body.Close()
	assert.Equal(t, http.StatusNoContent, deleteResp.StatusCode)
}

func TestProxyGroupWithoutScriptReturnsAllProxyNodes(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "fixtures", "provider_plain.yaml"))
	require.NoError(t, err)

	upstream := newFlakyUpstream(t, fixture)
	ts, _ := newTestServerWithRefreshAndGroups(t)
	defer ts.Close()

	providerID := createProvider(t, ts.URL, fmt.Sprintf(`{"name":"alpha","url":"%s"}`, upstream.server.URL))
	refreshResp := refreshProvider(t, ts.URL, providerID)
	defer refreshResp.Body.Close()
	require.Equal(t, http.StatusNoContent, refreshResp.StatusCode)

	createResp := postJSON(t, ts.URL+"/api/proxy-groups", `{"name":"All Nodes","script":""}`)
	defer createResp.Body.Close()
	require.Equal(t, http.StatusCreated, createResp.StatusCode)

	var created struct {
		Group struct {
			ID int64 `json:"id"`
		} `json:"group"`
	}
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&created))

	nodesResp, err := http.Get(fmt.Sprintf("%s/api/proxy-groups/%d/nodes", ts.URL, created.Group.ID))
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

func TestProxyGroupScriptFiltersNodesByReturnedIDs(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "fixtures", "provider_plain.yaml"))
	require.NoError(t, err)

	upstream := newFlakyUpstream(t, fixture)
	ts, _ := newTestServerWithRefreshAndGroups(t)
	defer ts.Close()

	providerID := createProvider(t, ts.URL, fmt.Sprintf(`{"name":"alpha","url":"%s"}`, upstream.server.URL))
	refreshResp := refreshProvider(t, ts.URL, providerID)
	defer refreshResp.Body.Close()
	require.Equal(t, http.StatusNoContent, refreshResp.StatusCode)

	createResp := postJSON(t, ts.URL+"/api/proxy-groups", `{"name":"Filtered","script":"function (proxyNodes) { return [proxyNodes[1].id] }"}`)
	defer createResp.Body.Close()
	require.Equal(t, http.StatusCreated, createResp.StatusCode)

	var created struct {
		Group struct {
			ID int64 `json:"id"`
		} `json:"group"`
	}
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&created))

	nodesResp, err := http.Get(fmt.Sprintf("%s/api/proxy-groups/%d/nodes", ts.URL, created.Group.ID))
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
	ts, _ := newTestServerWithRefreshAndGroups(t)
	defer ts.Close()

	providerID := createProvider(t, ts.URL, fmt.Sprintf(`{"name":"alpha","url":"%s"}`, upstream.server.URL))
	refreshResp := refreshProvider(t, ts.URL, providerID)
	defer refreshResp.Body.Close()
	require.Equal(t, http.StatusNoContent, refreshResp.StatusCode)

	createResp := postJSON(t, ts.URL+"/api/proxy-groups", `{"name":"Broken","script":"function (proxyNodes) { throw new Error('boom') }"}`)
	defer createResp.Body.Close()
	require.Equal(t, http.StatusCreated, createResp.StatusCode)

	var created struct {
		Group struct {
			ID int64 `json:"id"`
		} `json:"group"`
	}
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&created))

	nodesResp, err := http.Get(fmt.Sprintf("%s/api/proxy-groups/%d/nodes", ts.URL, created.Group.ID))
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

func TestUpdatingOneGroupScriptDoesNotChangeOtherGroups(t *testing.T) {
	ts := newTestServerWithGroups(t)
	defer ts.Close()

	firstResp := postJSON(t, ts.URL+"/api/proxy-groups", `{"name":"First","script":"function (proxyNodes) { return [] }"}`)
	defer firstResp.Body.Close()
	secondResp := postJSON(t, ts.URL+"/api/proxy-groups", `{"name":"Second","script":"function (proxyNodes) { return proxyNodes.map(function (node) { return node.id }) }"}`)
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

	updateResp := putJSON(t, fmt.Sprintf("%s/api/proxy-groups/%d", ts.URL, first.Group.ID), `{"name":"First","script":""}`)
	defer updateResp.Body.Close()
	require.Equal(t, http.StatusOK, updateResp.StatusCode)

	getResp, err := http.Get(fmt.Sprintf("%s/api/proxy-groups/%d", ts.URL, second.Group.ID))
	require.NoError(t, err)
	defer getResp.Body.Close()
	assert.Contains(t, readBody(t, getResp), "return node.id")
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

func TestRemovingScriptKeepsGroupValid(t *testing.T) {
	ts := newTestServerWithGroups(t)
	defer ts.Close()

	createResp := postJSON(t, ts.URL+"/api/proxy-groups", `{"name":"Optional","script":"function (proxyNodes) { return [] }"}`)
	defer createResp.Body.Close()

	var created struct {
		Group struct {
			ID int64 `json:"id"`
		} `json:"group"`
	}
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&created))

	updateResp := putJSON(t, fmt.Sprintf("%s/api/proxy-groups/%d", ts.URL, created.Group.ID), `{"name":"Optional","script":""}`)
	defer updateResp.Body.Close()
	require.Equal(t, http.StatusOK, updateResp.StatusCode)

	getResp, err := http.Get(fmt.Sprintf("%s/api/proxy-groups/%d", ts.URL, created.Group.ID))
	require.NoError(t, err)
	defer getResp.Body.Close()
	assert.Equal(t, http.StatusOK, getResp.StatusCode)
	assert.Contains(t, readBody(t, getResp), `"script":""`)
}
