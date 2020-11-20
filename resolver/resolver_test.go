package resolver

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	testResolvers = []string{
		"1.2.4.8",
		"8.8.8.8",
		"114.114.114.114",
	}

	testsDomains = []string{
		"baidu.com",
		"google.com",
		"qq.com",
	}
)

func TestResolver_NoFresh(t *testing.T) {
	require := require.New(t)
	for _, k := range testResolvers {
		r := NewResolver(k)
		for _, v := range testsDomains {
			r1, err := r.Get(v)
			t.Logf("%s => %s => %v", v, k, r1)
			require.NoError(err)

		}
	}
}
