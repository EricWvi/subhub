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

	apiMux := http.NewServeMux()
	handler.RegisterRoutes(apiMux)

	mux := http.NewServeMux()
	mux.Handle("/api/", http.StripPrefix("/api", apiMux))
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
	resp := postJSON(t, baseURL+"/api/providers", body)
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

	resp := postJSON(t, ts.URL+"/api/providers", `{"name":"alpha","url":"not-a-url"}`)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, readBody(t, resp), "invalid provider url")
}

func TestUpdateProviderRejectsInvalidURL(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	id := createProvider(t, ts.URL, `{"name":"alpha","url":"https://example.com/sub"}`)
	resp := putJSON(t, fmt.Sprintf("%s/api/providers/%d", ts.URL, id), `{"name":"beta","url":"not-a-url"}`)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, readBody(t, resp), "invalid provider url")
}

func TestListProvidersStartsEmpty(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/providers")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.JSONEq(t, `{"providers":[]}`, readBody(t, resp))
}

func TestCreateProviderUsesDefaultRefreshInterval(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	body := `{"name":"alpha","url":"https://example.com/sub"}`
	resp := postJSON(t, ts.URL+"/api/providers", body)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result map[string]map[string]any
	err := json.Unmarshal([]byte(readBody(t, resp)), &result)
	require.NoError(t, err)

	p := result["provider"]
	assert.Equal(t, "alpha", p["name"])
	assert.Equal(t, "https://example.com/sub", p["url"])
	assert.Equal(t, float64(120), p["refresh_interval_minutes"])
}

func TestCreateProviderReturnsPhase2MetadataFields(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp := postJSON(t, ts.URL+"/api/providers", `{"name":"alpha","url":"https://example.com/sub","abbrev":"HK"}`)
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var result struct {
		Provider struct {
			Abbrev string `json:"abbrev"`
			Used   int64  `json:"used"`
			Total  int64  `json:"total"`
			Expire int64  `json:"expire"`
		} `json:"provider"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "HK", result.Provider.Abbrev)
	assert.EqualValues(t, 0, result.Provider.Used)
	assert.EqualValues(t, 0, result.Provider.Total)
	assert.EqualValues(t, 0, result.Provider.Expire)
}

func TestCreateProviderUppercasesAbbrevAndAllowsDuplicates(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	first := postJSON(t, ts.URL+"/api/providers", `{"name":"alpha","url":"https://example.com/a","abbrev":"hk"}`)
	defer first.Body.Close()
	require.Equal(t, http.StatusCreated, first.StatusCode)

	second := postJSON(t, ts.URL+"/api/providers", `{"name":"beta","url":"https://example.com/b","abbrev":"HK"}`)
	defer second.Body.Close()
	require.Equal(t, http.StatusCreated, second.StatusCode)

	listResp, err := http.Get(ts.URL + "/api/providers")
	require.NoError(t, err)
	defer listResp.Body.Close()
	assert.Contains(t, readBody(t, listResp), `"abbrev":"HK"`)
}

func TestUpdateProviderRejectsNonLetterAbbrev(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	id := createProvider(t, ts.URL, `{"name":"alpha","url":"https://example.com/sub","abbrev":"HK"}`)
	resp := putJSON(t, fmt.Sprintf("%s/api/providers/%d", ts.URL, id), `{"name":"alpha","url":"https://example.com/sub","abbrev":"H1"}`)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, readBody(t, resp), "uppercase letters only")
}

func TestUpdateAndDeleteProvider(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	id := createProvider(t, ts.URL, `{"name":"alpha","url":"https://example.com/sub"}`)

	putResp := putJSON(t, fmt.Sprintf("%s/api/providers/%d", ts.URL, id), `{"name":"beta","url":"https://example.net/sub","refresh_interval_minutes":60}`)
	defer putResp.Body.Close()
	assert.Equal(t, http.StatusOK, putResp.StatusCode)

	var result map[string]map[string]any
	err := json.Unmarshal([]byte(readBody(t, putResp)), &result)
	require.NoError(t, err)
	assert.Equal(t, "beta", result["provider"]["name"])
	assert.Equal(t, float64(60), result["provider"]["refresh_interval_minutes"])

	delResp := deleteRequest(t, fmt.Sprintf("%s/api/providers/%d", ts.URL, id))
	defer delResp.Body.Close()
	assert.Equal(t, http.StatusNoContent, delResp.StatusCode)

	getResp, err := http.Get(fmt.Sprintf("%s/api/providers/%d", ts.URL, id))
	require.NoError(t, err)
	defer getResp.Body.Close()
	assert.Equal(t, http.StatusNotFound, getResp.StatusCode)
}

func TestDeleteProviderRejectsSubscriptionReference(t *testing.T) {
	ts := newTestServerWithSubscriptions(t)
	defer ts.Close()

	providerID := createProvider(t, ts.URL, `{"name":"alpha","url":"https://example.com/a"}`)
	groupID := createProxyGroup(t, ts.URL, `{"name":"Streaming","script":""}`)

	resp := postJSON(t, ts.URL+"/api/subscriptions/proxy-providers", fmt.Sprintf(`{
		"name":"Provider Export",
		"providers":[%d],
		"internal_proxy_group_id":%d
	}`, providerID, groupID))
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	deleteResp := deleteRequest(t, fmt.Sprintf("%s/api/providers/%d", ts.URL, providerID))
	defer deleteResp.Body.Close()
	assert.Equal(t, http.StatusConflict, deleteResp.StatusCode)
	assert.Contains(t, readBody(t, deleteResp), "subscription")
}
