package parse_test

import (
	"testing"

	"github.com/EricWvi/subhub/internal/group"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectNodeIDsReturnsAllNodesForEmptyScript(t *testing.T) {
	nodes := []group.ProxyNodeView{
		{ID: 11, ProviderName: "alpha", Name: "vmess-hk-01"},
		{ID: 12, ProviderName: "alpha", Name: "ss-jp-01"},
	}

	selected, err := group.SelectNodeIDs("", nodes)
	require.NoError(t, err)
	assert.Equal(t, []int64{11, 12}, selected)
}

func TestSelectNodeIDsRunsJavaScriptFunction(t *testing.T) {
	nodes := []group.ProxyNodeView{
		{ID: 11, ProviderName: "alpha", Name: "vmess-hk-01"},
		{ID: 12, ProviderName: "beta", Name: "ss-jp-01"},
	}

	selected, err := group.SelectNodeIDs(`function (proxyNodes) { return [proxyNodes[1].id] }`, nodes)
	require.NoError(t, err)
	assert.Equal(t, []int64{12}, selected)
}

func TestSelectNodeIDsRejectsNonFunctionScript(t *testing.T) {
	nodes := []group.ProxyNodeView{{ID: 11, ProviderName: "alpha", Name: "vmess-hk-01"}}

	_, err := group.SelectNodeIDs(`var x = 1`, nodes)
	require.Error(t, err)
}

func TestSelectNodeIDsRejectsUnknownIDs(t *testing.T) {
	nodes := []group.ProxyNodeView{{ID: 11, ProviderName: "alpha", Name: "vmess-hk-01"}}

	_, err := group.SelectNodeIDs(`function (proxyNodes) { return [999] }`, nodes)
	require.Error(t, err)
}
