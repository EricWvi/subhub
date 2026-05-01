package parse_test

import (
	"os"
	"testing"

	"github.com/EricWvi/subhub/internal/parse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fixtureBytes(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return data
}

func TestParseSubscriptionAcceptsPlainYAML(t *testing.T) {
	payload := fixtureBytes(t, "../../tests/fixtures/provider_plain.yaml")

	nodes, format, err := parse.DecodeAndNormalize(payload)
	require.NoError(t, err)
	assert.Equal(t, "yaml", format)
	assert.Len(t, nodes, 2)
	assert.Equal(t, "vmess-hk-01", nodes[0]["name"])
	assert.Equal(t, "vmess", nodes[0]["type"])
	assert.Equal(t, "hk.example.com", nodes[0]["server"])
}

func TestParseSubscriptionAcceptsBase64EncodedYAML(t *testing.T) {
	payload := fixtureBytes(t, "../../tests/fixtures/provider_base64.txt")

	nodes, format, err := parse.DecodeAndNormalize(payload)
	require.NoError(t, err)
	assert.Equal(t, "base64+yaml", format)
	assert.Len(t, nodes, 2)
	assert.Equal(t, "vmess-hk-01", nodes[0]["name"])
	assert.Equal(t, "ss-jp-01", nodes[1]["name"])
}

func TestParseSubscriptionRejectsGarbage(t *testing.T) {
	_, _, err := parse.DecodeAndNormalize([]byte("not valid yaml at all {{{"))
	assert.Error(t, err)
}

func TestParseSubscriptionEmptyPayload(t *testing.T) {
	nodes, format, err := parse.DecodeAndNormalize([]byte(""))
	require.NoError(t, err)
	assert.Equal(t, "yaml", format)
	assert.Len(t, nodes, 0)
}
