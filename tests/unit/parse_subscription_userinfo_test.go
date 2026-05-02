package parse_test

import (
	"testing"

	"github.com/EricWvi/subhub/internal/fetch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSubscriptionUserinfo(t *testing.T) {
	meta, ok := fetch.ParseSubscriptionUserinfo("upload=1; download=2; total=3; expire=1710000000")
	require.True(t, ok)
	assert.EqualValues(t, 3, meta.Used)
	assert.EqualValues(t, 3, meta.Total)
	assert.EqualValues(t, 1710000000, meta.Expire)
}
