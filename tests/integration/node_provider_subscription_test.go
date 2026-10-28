package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/EricWvi/subhub/internal/subscription"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestNodeProviderSubscriptionLifecycleAndContent(t *testing.T) {
	ts := newTestServerWithSubscriptions(t)
	defer ts.Close()
	url := ts.URL + "/api/subscriptions/node-providers"

	resp, err := http.Get(url)
	require.NoError(t, err)
	assert.JSONEq(t, `{"subscriptions":[]}`, readBody(t, resp))
	resp.Body.Close()

	first := createCustomNode(t, ts.URL, "Office", "type: socks5\nserver: office.example.com\nport: 1080\n")
	second := createCustomNode(t, ts.URL, "Home", "type: http\nserver: home.example.com\nport: 8080\n")
	createCustomNode(t, ts.URL, "Excluded", "type: direct\n")
	resp = postJSON(t, url, fmt.Sprintf(`{"name":"My Nodes","nodes":[%d,%d]}`, second, first))
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var result struct {
		Subscription subscription.NodeProviderSubscription `json:"subscription"`
	}
	require.NoError(t, json.Unmarshal([]byte(readBody(t, resp)), &result))
	resp.Body.Close()
	assert.Equal(t, []int64{second, first}, result.Subscription.Nodes)
	_, offset := result.Subscription.CreatedAt.Zone()
	assert.Equal(t, 8*60*60, offset)
	itemURL := fmt.Sprintf("%s/%d", url, result.Subscription.ID)

	for _, endpoint := range []string{url, itemURL} {
		resp, err = http.Get(endpoint)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, readBody(t, resp), fmt.Sprintf(`"nodes":[%d,%d]`, second, first))
		resp.Body.Close()
	}

	resp, err = http.Get(itemURL + "/content")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/yaml", resp.Header.Get("Content-Type"))
	assert.Empty(t, resp.Header.Get("Subscription-Userinfo"))
	var output map[string][]map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(readBody(t, resp)), &output))
	resp.Body.Close()
	require.Len(t, output["proxies"], 2)
	assert.Equal(t, "Home", output["proxies"][0]["name"])
	assert.Equal(t, "Office", output["proxies"][1]["name"])

	resp = putJSON(t, ts.URL+fmt.Sprintf("/api/nodes/%d", first), `{"name":"Office Updated","config":"type: socks5\nserver: updated.example.com\nport: 1080\n"}`)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
	resp = putJSON(t, itemURL, fmt.Sprintf(`{"name":"Updated","nodes":[%d]}`, first))
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, readBody(t, resp), `"name":"Updated"`)
	resp.Body.Close()
	resp, err = http.Get(itemURL + "/content")
	require.NoError(t, err)
	body := readBody(t, resp)
	resp.Body.Close()
	assert.Contains(t, body, "Office Updated")
	assert.Contains(t, body, "updated.example.com")
	assert.NotContains(t, body, "Home")

	req, err := http.NewRequest(http.MethodDelete, ts.URL+fmt.Sprintf("/api/nodes/%d", first), nil)
	require.NoError(t, err)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()
	resp, err = http.Get(itemURL)
	require.NoError(t, err)
	assert.Contains(t, readBody(t, resp), `"nodes":[]`)
	resp.Body.Close()
	resp, err = http.Get(itemURL + "/content")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, readBody(t, resp), "type: direct")
	resp.Body.Close()

	req, err = http.NewRequest(http.MethodDelete, itemURL, nil)
	require.NoError(t, err)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()
	resp, err = http.Get(itemURL)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()
}

func TestNodeProviderSubscriptionValidation(t *testing.T) {
	ts := newTestServerWithSubscriptions(t)
	defer ts.Close()
	id := createCustomNode(t, ts.URL, "Office", "type: direct\n")
	url := ts.URL + "/api/subscriptions/node-providers"
	resp := postJSON(t, url, fmt.Sprintf(`{"name":"Valid","nodes":[%d]}`, id))
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	for _, body := range []string{
		fmt.Sprintf(`{"name":" ","nodes":[%d]}`, id),
		`{"name":"Empty","nodes":[]}`,
		`{"name":"Missing"}`,
		`{"name":"Unknown","nodes":[9999]}`,
		`{"name":"Invalid","nodes":[-1]}`,
		fmt.Sprintf(`{"name":"Duplicate","nodes":[%d,%d]}`, id, id),
		`{`,
	} {
		t.Run(body, func(t *testing.T) {
			for _, update := range []bool{false, true} {
				if update {
					resp = putJSON(t, url+"/1", body)
				} else {
					resp = postJSON(t, url, body)
				}
				assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
				resp.Body.Close()
			}
		})
	}
	resp, err := http.Get(url + "/1")
	require.NoError(t, err)
	assert.Contains(t, readBody(t, resp), `"name":"Valid"`)
	resp.Body.Close()
	resp = putJSON(t, url+"/9999", fmt.Sprintf(`{"name":"Missing","nodes":[%d]}`, id))
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()
	for _, path := range []string{"/9999/content", "/1/content/extra"} {
		resp, err = http.Get(url + path)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		resp.Body.Close()
	}
}
