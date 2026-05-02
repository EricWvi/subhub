package parse_test

import (
	"strings"
	"testing"

	"github.com/EricWvi/subhub/internal/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderMihomoInjectsNormalizedNodesIntoTemplate(t *testing.T) {
	nodes := []map[string]any{
		{"name": "vmess-hk-01", "type": "vmess", "server": "hk.example.com", "port": 443},
		{"name": "ss-jp-01", "type": "ss", "server": "jp.example.com", "port": 8388},
	}

	out, err := render.MihomoTemplate("../../tests/fixtures/template.yaml", nodes, nil)
	require.NoError(t, err)
	assert.Contains(t, out, "proxies:")
	assert.Contains(t, out, "name: vmess-hk-01")
	assert.Contains(t, out, "name: ss-jp-01")
}

func TestRenderMihomoPreservesProxyGroupsAndRules(t *testing.T) {
	nodes := []map[string]any{
		{"name": "test-node", "type": "vmess", "server": "test.example.com", "port": 443},
	}

	out, err := render.MihomoTemplate("../../tests/fixtures/template.yaml", nodes, nil)
	require.NoError(t, err)
	assert.Contains(t, out, "proxy-groups:")
	assert.Contains(t, out, "rules:")
	assert.Contains(t, out, "DOMAIN-SUFFIX,google.com,PROXY")
}

func TestRenderMihomoReturnsErrorForMissingTemplate(t *testing.T) {
	_, err := render.MihomoTemplate("nonexistent.yaml", nil, nil)
	assert.Error(t, err)
}

func TestRenderMihomoPrependsManualRulesBeforeTemplateRules(t *testing.T) {
	nodes := []map[string]any{
		{"name": "test-node", "type": "vmess", "server": "test.example.com", "port": 443},
	}
	rules := []string{
		"DOMAIN-SUFFIX,openai.com,DIRECT",
		"DOMAIN-KEYWORD,netflix,REJECT",
	}

	out, err := render.MihomoTemplate("../../tests/fixtures/template.yaml", nodes, rules)
	require.NoError(t, err)

	firstManual := strings.Index(out, "DOMAIN-SUFFIX,openai.com,DIRECT")
	firstTemplate := strings.Index(out, "DOMAIN-SUFFIX,google.com,PROXY")
	assert.NotEqual(t, -1, firstManual)
	assert.NotEqual(t, -1, firstTemplate)
	assert.Less(t, firstManual, firstTemplate)
}
