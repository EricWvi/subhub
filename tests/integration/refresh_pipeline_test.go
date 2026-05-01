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
	"github.com/EricWvi/subhub/internal/provider"
	"github.com/EricWvi/subhub/internal/refresh"
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

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
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
	resp, err := http.Post(fmt.Sprintf("%s/providers/%d/refresh", baseURL, providerID), "application/json", nil)
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
	require.True(t, firstSnap.IsLastKnownGood)
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
	assert.Contains(t, string(snap.RawPayload), "vmess-hk-01")
}

func TestRefreshReturnsNotFoundForMissingProvider(t *testing.T) {
	ts, _ := newTestServerWithRefresh(t)
	defer ts.Close()

	resp := refreshProvider(t, ts.URL, 99999)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
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
