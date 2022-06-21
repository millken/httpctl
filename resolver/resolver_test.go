package resolver

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	testResolvers = []string{
		"1.2.4.8",
		"8.8.8.8:53",
		"9.9.9.9:9953",
	}

	testsDomains = []string{
		"baidu.com",
		"google.com",
		"github.com",
	}
)

func TestResolver_NoFresh(t *testing.T) {
	require := require.New(t)
	r := NewResolver(testResolvers...)
	for _, v := range testsDomains {
		r1, n, err := r.Lookup(v)
		t.Logf("%s => %s => %v", v, n, r1)
		require.NoError(err)

	}
}
