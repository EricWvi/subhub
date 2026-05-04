package parse_test

import (
	"strings"
	"testing"

	"github.com/EricWvi/subhub/internal/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderClashConfigSubscriptionAssemblesYAML(t *testing.T) {
	proxies := []map[string]any{
		{"name": "hk-01", "type": "vmess", "server": "hk.example.com", "port": 443},
		{"name": "jp-01", "type": "ss", "server": "jp.example.com", "port": 8388},
	}
	groups := []render.RenderedProxyGroup{
		{Name: "Proxies", Type: "select", Proxies: []string{"hk-01", "jp-01", "DIRECT"}},
		{Name: "Auto", Type: "url-test", URL: "http://www.gstatic.com/generate_204", Interval: 300, Proxies: []string{"hk-01", "jp-01"}},
	}

	out, err := render.RenderClashConfigSubscription("../../tests/fixtures/client_sub.yaml", proxies, groups, nil)
	require.NoError(t, err)
	assert.Contains(t, out, "name: hk-01")
	assert.Contains(t, out, "name: jp-01")
	assert.Contains(t, out, "name: Proxies")
	assert.Contains(t, out, "name: Auto")
	assert.Contains(t, out, "type: select")
	assert.Contains(t, out, "type: url-test")
	assert.Contains(t, out, "url: http://www.gstatic.com/generate_204")
}

func TestRenderClashConfigSubscriptionPrependsManualRules(t *testing.T) {
	proxies := []map[string]any{
		{"name": "test", "type": "vmess", "server": "test.example.com", "port": 443},
	}
	groups := []render.RenderedProxyGroup{
		{Name: "Proxies", Type: "select", Proxies: []string{"DIRECT"}},
	}
	rules := []string{
		"DOMAIN-SUFFIX,openai.com,Proxies",
		"DOMAIN-KEYWORD,netflix,REJECT",
	}

	out, err := render.RenderClashConfigSubscription("../../tests/fixtures/client_sub.yaml", proxies, groups, rules)
	require.NoError(t, err)

	firstManual := strings.Index(out, "DOMAIN-SUFFIX,openai.com,Proxies")
	firstTemplate := strings.Index(out, "GEOIP,CN,DIRECT")
	assert.NotEqual(t, -1, firstManual)
	assert.NotEqual(t, -1, firstTemplate)
	assert.Less(t, firstManual, firstTemplate)
}

func TestRenderClashConfigSubscriptionReturnsErrorForMissingTemplate(t *testing.T) {
	_, err := render.RenderClashConfigSubscription("nonexistent.yaml", nil, nil, nil)
	assert.Error(t, err)
}

func TestRenderClashConfigSubscriptionWithEmptyGroups(t *testing.T) {
	proxies := []map[string]any{
		{"name": "test", "type": "vmess", "server": "test.example.com", "port": 443},
	}

	out, err := render.RenderClashConfigSubscription("../../tests/fixtures/client_sub.yaml", proxies, nil, nil)
	require.NoError(t, err)
	assert.Contains(t, out, "name: test")
	assert.Contains(t, out, "proxy-groups:")
}

func TestRenderProxyProviderSubscription(t *testing.T) {
	nodes := []map[string]any{
		{"name": "hk-01", "type": "vmess", "server": "hk.example.com", "port": 443},
	}

	out, err := render.RenderProxyProviderSubscription(nodes)
	require.NoError(t, err)
	assert.Contains(t, out, "proxies:")
	assert.Contains(t, out, "name: hk-01")
}

func TestRenderProxyProviderSubscriptionEmpty(t *testing.T) {
	out, err := render.RenderProxyProviderSubscription(nil)
	require.NoError(t, err)
	assert.Contains(t, out, "proxies:")
}

func TestRenderRuleProviderSubscription(t *testing.T) {
	rules := []string{
		"DOMAIN-SUFFIX,google.com,PROXY",
		"DOMAIN-KEYWORD,netflix,REJECT",
	}

	out, err := render.RenderRuleProviderSubscription(rules)
	require.NoError(t, err)
	assert.Contains(t, out, "payload:")
	assert.Contains(t, out, "DOMAIN-SUFFIX,google.com")
	assert.Contains(t, out, "DOMAIN-KEYWORD,netflix")
}

func TestRenderRuleProviderSubscriptionEmpty(t *testing.T) {
	out, err := render.RenderRuleProviderSubscription(nil)
	require.NoError(t, err)
	assert.Contains(t, out, "payload:")
}

func TestRenderClashConfigSubscriptionPreservesFinalGroup(t *testing.T) {
	proxies := []map[string]any{
		{"name": "test", "type": "vmess", "server": "test.example.com", "port": 443},
	}
	groups := []render.RenderedProxyGroup{
		{Name: "Proxies", Type: "select", Proxies: []string{"test", "DIRECT"}},
	}

	out, err := render.RenderClashConfigSubscription("../../tests/fixtures/client_sub.yaml", proxies, groups, nil)
	require.NoError(t, err)
	assert.Contains(t, out, "name: Final")
	assert.Contains(t, out, "name: Proxies")
	assert.Contains(t, out, "MATCH,Final")
}
