package proxy

import (
	"context"
	"net"
	"sync"
	"time"
)

var (
	DefaultExpiration time.Duration = time.Minute * 10
	ResolverTimeout   time.Duration = time.Second * 7
)

type Item struct {
	Object     []string
	Expiration int64
}

// Returns true if the item has expired.
func (item Item) Expired() bool {
	if item.Expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > item.Expiration
}

type Resolver struct {
	sync.RWMutex
	ctx      context.Context
	resolver string
	cache    map[string]Item
}

func NewResolver(resolver string, refreshRate time.Duration) *Resolver {
	r := &Resolver{
		ctx:      context.Background(),
		resolver: resolver,
		cache:    make(map[string]Item),
	}
	if refreshRate > 0 {
		go r.autoRefresh(refreshRate)
	}
	return r
}

func (r *Resolver) WithContext(ctx context.Context) *Resolver {
	r.ctx = ctx
	return r
}

func (r *Resolver) Get(host string) ([]string, error) {
	r.RLock()
	item, found := r.cache[host]
	if found && !item.Expired() {
		r.RUnlock()
		return item.Object, nil
	}
	r.RUnlock()

	ctx, _ := context.WithTimeout(r.ctx, ResolverTimeout)
	return r.Lookup(ctx, host)
}

func (r *Resolver) get(host string) (ips []string, isCached bool, cacheExpiration int64) {
	r.RLock()
	item, isCached := r.cache[host]
	if isCached {
		ips = item.Object
		cacheExpiration = item.Expiration
	}
	r.RUnlock()
	return
}

func (r *Resolver) Refresh() {
	r.Lock()
	for k, v := range r.cache {
		if v.Expired() {
			delete(r.cache, k)
		}
	}
	r.Unlock()

	i := 0
	r.RLock()
	addresses := make([]string, len(r.cache))
	for key, _ := range r.cache {
		addresses[i] = key
		i++
	}
	r.RUnlock()

	for _, host := range addresses {
		ctx, _ := context.WithTimeout(r.ctx, ResolverTimeout)
		r.Lookup(ctx, host)
	}
}

func (r *Resolver) Lookup(ctx context.Context, host string) ([]string, error) {
	rr := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: time.Millisecond * time.Duration(10000),
			}
			return d.DialContext(ctx, "udp", r.resolver+":53")
		},
	}
	ips, err := rr.LookupHost(ctx, host)
	if err != nil {
		return nil, err
	}

	r.Lock()
	r.cache[host] = Item{ips, time.Now().Add(DefaultExpiration).UnixNano()}
	r.Unlock()
	return ips, nil
}

func (r *Resolver) autoRefresh(rate time.Duration) {
	timer := time.NewTimer(rate)
	for {
		select {
		case <-r.ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			r.Refresh()
			timer.Reset(rate)
		}
	}
}
