package integration

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServerWithRefresh(t *testing.T) (*httptest.Server, *provider.Repository) {
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
	outputHandler := output.NewHandler(repo, nil, templatePath)

	apiMux := http.NewServeMux()
	handler.RegisterRoutes(apiMux)
	outputHandler.RegisterRoutes(apiMux)

	mux := http.NewServeMux()
	mux.Handle("/api/", http.StripPrefix("/api", apiMux))
	return httptest.NewServer(mux), repo
}

type flakyUpstream struct {
	server   *httptest.Server
	fail     atomic.Bool
	failCode int
	failMsg  string
	payload  []byte
}

func newFlakyUpstream(t *testing.T, payload []byte) *flakyUpstream {
	t.Helper()
	fu := &flakyUpstream{payload: payload, failCode: http.StatusBadGateway, failMsg: "upstream down"}
	fu.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fu.fail.Load() {
			http.Error(w, fu.failMsg, fu.failCode)
			return
		}
		w.Header().Set("Content-Type", "text/yaml")
		w.Write(fu.payload)
	}))
	t.Cleanup(fu.server.Close)
	return fu
}

func refreshProvider(t *testing.T, baseURL string, providerID int64) *http.Response {
	t.Helper()
	resp, err := http.Post(fmt.Sprintf("%s/api/providers/%d/refresh", baseURL, providerID), "application/json", nil)
	require.NoError(t, err)
	return resp
}

func TestRefreshRetainsPreviousSnapshotWhenProviderFails(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "fixtures", "provider_plain.yaml"))
	require.NoError(t, err)

	upstream := newFlakyUpstream(t, fixture)
	ts, repo := newTestServerWithRefresh(t)
	defer ts.Close()

	providerID := createProvider(t, ts.URL, fmt.Sprintf(`{"name":"alpha","url":"%s"}`, upstream.server.URL))

	firstResp := refreshProvider(t, ts.URL, providerID)
	defer firstResp.Body.Close()
	require.Equal(t, http.StatusNoContent, firstResp.StatusCode)

	ctx := context.Background()
	firstSnap, err := repo.GetLatestSnapshot(ctx, providerID)
	require.NoError(t, err)
	require.Equal(t, 2, firstSnap.NodeCount)

	upstream.fail.Store(true)

	secondResp := refreshProvider(t, ts.URL, providerID)
	defer secondResp.Body.Close()
	assert.Equal(t, http.StatusBadGateway, secondResp.StatusCode)

	secondSnap, err := repo.GetLatestSnapshot(ctx, providerID)
	require.NoError(t, err)
	assert.Equal(t, firstSnap.ID, secondSnap.ID)
	assert.Equal(t, firstSnap.NormalizedYAML, secondSnap.NormalizedYAML)
}

func TestRefreshSucceedsOnFirstFetch(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "fixtures", "provider_plain.yaml"))
	require.NoError(t, err)

	upstream := newFlakyUpstream(t, fixture)
	ts, repo := newTestServerWithRefresh(t)
	defer ts.Close()

	providerID := createProvider(t, ts.URL, fmt.Sprintf(`{"name":"beta","url":"%s"}`, upstream.server.URL))

	resp := refreshProvider(t, ts.URL, providerID)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	ctx := context.Background()
	snap, err := repo.GetLatestSnapshot(ctx, providerID)
	require.NoError(t, err)
	assert.Equal(t, "yaml", snap.Format)
	assert.Equal(t, 2, snap.NodeCount)
	assert.Contains(t, snap.NormalizedYAML, "vmess-hk-01")
}

func TestRefreshReturnsNotFoundForMissingProvider(t *testing.T) {
	ts, _ := newTestServerWithRefresh(t)
	defer ts.Close()

	resp := refreshProvider(t, ts.URL, 99999)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

type fakeClock struct {
	now time.Time
}

func (c *fakeClock) Now() time.Time { return c.now }

func TestSchedulerRefreshesDueProvidersOnly(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "scheduler_test.db")
	db := store.MustOpen(dbPath)
	defer db.Close()

	repo := provider.NewRepository(db)
	ctx := context.Background()

	p1, err := repo.Create(ctx, provider.Provider{Name: "due", URL: "https://due.example.com", RefreshIntervalMinutes: 5})
	require.NoError(t, err)

	p2, err := repo.Create(ctx, provider.Provider{Name: "fresh", URL: "https://fresh.example.com", RefreshIntervalMinutes: 120})
	require.NoError(t, err)

	fakeNow := time.Now().UTC()
	_, err = db.ExecContext(ctx, `UPDATE providers SET updated_at = ? WHERE id = ?`, fakeNow.Add(-10*time.Minute).Format(time.RFC3339), p1.ID)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `UPDATE providers SET updated_at = ? WHERE id = ?`, fakeNow.Add(-30*time.Minute).Format(time.RFC3339), p2.ID)
	require.NoError(t, err)

	var refreshed []int64
	countFn := func(ctx context.Context, id int64) error {
		refreshed = append(refreshed, id)
		return nil
	}

	scheduler := refresh.NewScheduler(repo, countFn, time.Minute)
	scheduler.WithClock(&fakeClock{now: fakeNow})
	scheduler.RunOnce(ctx)

	assert.Equal(t, []int64{p1.ID}, refreshed)
}

func TestRefreshRecordsFailureOnUpstreamError(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "fixtures", "provider_plain.yaml"))
	require.NoError(t, err)

	upstream := newFlakyUpstream(t, fixture)
	ts, _ := newTestServerWithRefresh(t)
	defer ts.Close()

	providerID := createProvider(t, ts.URL, fmt.Sprintf(`{"name":"gamma","url":"%s"}`, upstream.server.URL))

	upstream.fail.Store(true)
	resp := refreshProvider(t, ts.URL, providerID)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadGateway, resp.StatusCode)
}

func TestMihomoOutputContainsNodesFromAllProviders(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "fixtures", "provider_plain.yaml"))
	require.NoError(t, err)

	upstream := newFlakyUpstream(t, fixture)
	ts, _ := newTestServerWithRefresh(t)
	defer ts.Close()

	providerID := createProvider(t, ts.URL, fmt.Sprintf(`{"name":"delta","url":"%s"}`, upstream.server.URL))

	refreshResp := refreshProvider(t, ts.URL, providerID)
	defer refreshResp.Body.Close()
	require.Equal(t, http.StatusNoContent, refreshResp.StatusCode)

	mihomoResp, err := http.Get(ts.URL + "/api/subscriptions/mihomo")
	require.NoError(t, err)
	defer mihomoResp.Body.Close()
	assert.Equal(t, http.StatusOK, mihomoResp.StatusCode)
	assert.Equal(t, "application/yaml", mihomoResp.Header.Get("Content-Type"))

	body := readBody(t, mihomoResp)
	assert.Contains(t, body, "name: vmess-hk-01")
	assert.Contains(t, body, "name: ss-jp-01")
	assert.Contains(t, body, "proxy-groups:")
	assert.Contains(t, body, "rules:")
}

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

	mihomoResp, err := http.Get(ts.URL + "/api/subscriptions/mihomo")
	require.NoError(t, err)
	defer mihomoResp.Body.Close()

	body := readBody(t, mihomoResp)
	assert.Contains(t, body, "name: vmess-hk-01")
	assert.Contains(t, body, "name: ss-jp-01")
}

func TestMihomoOutputEmptyWhenNoSnapshots(t *testing.T) {
	ts, _ := newTestServerWithRefresh(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/subscriptions/mihomo")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body := readBody(t, resp)
	assert.Contains(t, body, "proxies: []")
}

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
	ruleHandler := rule.NewHandler(rule.NewService(ruleRepo))
	ruleHandler.RegisterRoutes(apiMux)
	outputHandler.RegisterRoutes(apiMux)

	mux := http.NewServeMux()
	mux.Handle("/api/", http.StripPrefix("/api", apiMux))
	return httptest.NewServer(mux), providerRepo
}

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
