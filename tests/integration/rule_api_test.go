package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/EricWvi/subhub/internal/config"
	"github.com/EricWvi/subhub/internal/group"
	"github.com/EricWvi/subhub/internal/provider"
	"github.com/EricWvi/subhub/internal/rule"
	"github.com/EricWvi/subhub/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServerWithRules(t *testing.T) *httptest.Server {
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

	ruleRepo := rule.NewRepository(db)
	ruleSvc := rule.NewService(ruleRepo)
	ruleHandler := rule.NewHandler(ruleSvc)

	apiMux := http.NewServeMux()
	handler.RegisterRoutes(apiMux)
	groupHandler.RegisterRoutes(apiMux)
	ruleHandler.RegisterRoutes(apiMux)

	mux := http.NewServeMux()
	mux.Handle("/api/", http.StripPrefix("/api", apiMux))
	return httptest.NewServer(mux)
}

func createProxyGroup(t *testing.T, baseURL, body string) int64 {
	t.Helper()
	resp := postJSON(t, baseURL+"/api/proxy-groups", body)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var result struct {
		Group struct {
			ID int64 `json:"id"`
		} `json:"group"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	return result.Group.ID
}

func TestListRulesStartsEmpty(t *testing.T) {
	ts := newTestServerWithRules(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/rules")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.JSONEq(t, `{"rules":[],"page":1,"page_size":20,"total":0}`, readBody(t, resp))
}

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
