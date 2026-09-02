package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createCustomNode(t *testing.T, baseURL, name, config string) int64 {
	t.Helper()
	body, err := json.Marshal(map[string]string{"name": name, "config": config})
	require.NoError(t, err)
	resp := postJSON(t, baseURL+"/api/nodes", string(body))
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var result struct {
		Node struct {
			ID int64 `json:"id"`
		} `json:"node"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	return result.Node.ID
}

func TestCustomNodeCRUD(t *testing.T) {
	ts := newTestServerWithSubscriptions(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/nodes")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.JSONEq(t, `{"nodes":[]}`, readBody(t, resp))
	resp.Body.Close()

	nodeID := createCustomNode(t, ts.URL, "Office", "type: socks5\nserver: office.example.com\nport: 1080\n")
	updateBody := `{"name":"Office Updated","config":"type: http\nserver: office.example.com\nport: 8080\n"}`
	updateResp := putJSON(t, fmt.Sprintf("%s/api/nodes/%d", ts.URL, nodeID), updateBody)
	require.Equal(t, http.StatusOK, updateResp.StatusCode)
	assert.Contains(t, readBody(t, updateResp), `"name":"Office Updated"`)
	updateResp.Body.Close()

	deleteResp := deleteRequest(t, fmt.Sprintf("%s/api/nodes/%d", ts.URL, nodeID))
	assert.Equal(t, http.StatusNoContent, deleteResp.StatusCode)
	deleteResp.Body.Close()
}

func TestCustomNodeRejectsInvalidConfig(t *testing.T) {
	ts := newTestServerWithSubscriptions(t)
	defer ts.Close()

	resp := postJSON(t, ts.URL+"/api/nodes", `{"name":"Broken","config":"type: ["}`)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, readBody(t, resp), "valid YAML object")
}

func TestClashConfigRejectsDuplicateCustomNodeInOneGroup(t *testing.T) {
	ts := newTestServerWithSubscriptions(t)
	defer ts.Close()

	providerID := createProvider(t, ts.URL, `{"name":"alpha","url":"https://example.com/a"}`)
	nodeID := createCustomNode(t, ts.URL, "Office", "type: socks5\nserver: office.example.com\nport: 1080\n")
	resp := postJSON(t, ts.URL+"/api/subscriptions/clash-configs", fmt.Sprintf(`{
		"name":"Daily",
		"providers":[%d],
		"proxy_groups":[{
			"name":"Proxies",
			"type":"select",
			"proxies":[
				{"type":"custom","value":"%d"},
				{"type":"custom","value":"%d"}
			]
		}]
	}`, providerID, nodeID, nodeID))
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, readBody(t, resp), "only be selected once")
}
