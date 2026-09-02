package integration

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClashConfigContentIncludesCustomNodesUsedByGroups(t *testing.T) {
	ts, _ := newTestServerWithSubscriptionOutput(t)
	defer ts.Close()

	providerID := createProvider(t, ts.URL, `{"name":"alpha","url":"https://example.com/a"}`)
	nodeID := createCustomNode(t, ts.URL, "Office", "type: socks5\nserver: office.example.com\nport: 1080\n")

	createResp := postJSON(t, ts.URL+"/api/subscriptions/clash-configs", fmt.Sprintf(`{
		"name":"Custom Nodes",
		"providers":[%d],
		"proxy_groups":[{
			"name":"Proxies",
			"type":"select",
			"proxies":[{"type":"custom","value":"%d"}]
		}]
	}`, providerID, nodeID))
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	createResp.Body.Close()

	contentResp, err := http.Get(ts.URL + "/api/subscriptions/clash-configs/1/content")
	require.NoError(t, err)
	defer contentResp.Body.Close()

	require.Equal(t, http.StatusOK, contentResp.StatusCode)
	body := readBody(t, contentResp)
	assert.Contains(t, body, "name: Office")
	assert.Contains(t, body, "server: office.example.com")
	assert.Contains(t, body, "- Office")
}
