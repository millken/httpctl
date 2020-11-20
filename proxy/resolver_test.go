package proxy

import (
	"testing"
	"time"

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
		r := NewResolver(k, 0)
		for _, v := range testsDomains {
			r1, err := r.Get(v)
			t.Logf("%s => %s => %v", v, k, r1)
			require.NoError(err)

		}
	}
}

func TestResolver_WithCache(t *testing.T) {
	require := require.New(t)
	k := testResolvers[1]
	v := testsDomains[1]
	DefaultExpiration = time.Second * 1

	r := NewResolver(k, 2*time.Second)
	r1, err := r.Get(v)
	t.Logf("%s => %s => %v", v, k, r1)
	require.NoError(err)

	r2, isCached, exp := r.get(v)
	t.Logf("%s => %s => %v[%v, %d]", v, k, r2, isCached, exp)
	require.Greater(len(r2), 0)
	require.True(isCached)
	require.Greater(exp, int64(0))

	time.Sleep(3 * time.Second)

	r2, isCached, exp = r.get(v)
	t.Logf("%s => %s => %v[%v, %d]", v, k, r2, isCached, exp)
	require.Equal(0, len(r2))
	require.False(isCached)
	require.Equal(exp, int64(0))
}
