package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/EricWvi/subhub/internal/config"
	"github.com/EricWvi/subhub/internal/provider"
	"github.com/EricWvi/subhub/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T) *httptest.Server {
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

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	return httptest.NewServer(mux)
}

func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return string(b)
}

func postJSON(t *testing.T, url, body string) *http.Response {
	t.Helper()
	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	require.NoError(t, err)
	return resp
}

func putJSON(t *testing.T, url, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPut, url, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

func deleteRequest(t *testing.T, url string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

func createProvider(t *testing.T, baseURL, body string) int64 {
	t.Helper()
	resp := postJSON(t, baseURL+"/providers", body)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var result struct {
		Provider struct {
			ID int64 `json:"id"`
		} `json:"provider"`
	}
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	return result.Provider.ID
}

func TestCreateProviderRejectsInvalidURL(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp := postJSON(t, ts.URL+"/providers", `{"name":"alpha","url":"not-a-url"}`)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, readBody(t, resp), "invalid provider url")
}

func TestUpdateProviderRejectsInvalidURL(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	id := createProvider(t, ts.URL, `{"name":"alpha","url":"https://example.com/sub"}`)
	resp := putJSON(t, fmt.Sprintf("%s/providers/%d", ts.URL, id), `{"name":"beta","url":"not-a-url"}`)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, readBody(t, resp), "invalid provider url")
}

func TestListProvidersStartsEmpty(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/providers")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.JSONEq(t, `{"providers":[]}`, readBody(t, resp))
}

func TestCreateProviderUsesDefaultRefreshInterval(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	body := `{"name":"alpha","url":"https://example.com/sub"}`
	resp := postJSON(t, ts.URL+"/providers", body)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result map[string]map[string]any
	err := json.Unmarshal([]byte(readBody(t, resp)), &result)
	require.NoError(t, err)

	p := result["provider"]
	assert.Equal(t, "alpha", p["name"])
	assert.Equal(t, "https://example.com/sub", p["url"])
	assert.Equal(t, float64(7200), p["refresh_interval_seconds"])
}

func TestUpdateAndDeleteProvider(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	id := createProvider(t, ts.URL, `{"name":"alpha","url":"https://example.com/sub"}`)

	putResp := putJSON(t, fmt.Sprintf("%s/providers/%d", ts.URL, id), `{"name":"beta","url":"https://example.net/sub","refresh_interval_seconds":3600}`)
	defer putResp.Body.Close()
	assert.Equal(t, http.StatusOK, putResp.StatusCode)

	var result map[string]map[string]any
	err := json.Unmarshal([]byte(readBody(t, putResp)), &result)
	require.NoError(t, err)
	assert.Equal(t, "beta", result["provider"]["name"])
	assert.Equal(t, float64(3600), result["provider"]["refresh_interval_seconds"])

	delResp := deleteRequest(t, fmt.Sprintf("%s/providers/%d", ts.URL, id))
	defer delResp.Body.Close()
	assert.Equal(t, http.StatusNoContent, delResp.StatusCode)

	getResp, err := http.Get(fmt.Sprintf("%s/providers/%d", ts.URL, id))
	require.NoError(t, err)
	defer getResp.Body.Close()
	assert.Equal(t, http.StatusNotFound, getResp.StatusCode)
}
