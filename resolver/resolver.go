package resolver

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/miekg/dns"
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

func NewResolver(resolver string) *Resolver {
	r := &Resolver{
		ctx:      context.Background(),
		resolver: resolver,
		cache:    make(map[string]Item),
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
	return r.lookupHost(ctx, host)
}

func (r *Resolver) deleteExpired() {
	r.Lock()
	for k, v := range r.cache {
		if v.Expired() {
			delete(r.cache, k)
		}
	}
	r.Unlock()
}

func (r *Resolver) Refresh() {
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
		r.lookupHost(ctx, host)
	}
}

func (r *Resolver) lookupHost(ctx context.Context, host string) ([]string, error) {
	m1 := new(dns.Msg)
	m1.Id = dns.Id()
	m1.RecursionDesired = true
	m1.Question = make([]dns.Question, 1)
	m1.Question[0] = dns.Question{dns.Fqdn(host), dns.TypeA, dns.ClassINET}

	c := new(dns.Client)
	in, _, err := c.ExchangeContext(ctx, m1, r.resolver+":53")

	if err != nil {
		return nil, err
	}

	l := len(in.Answer)
	if l == 0 {
		return nil, errors.New(" answer has empty")
	}
	ips := make([]string, l)
	for i := 0; i < l; i++ {
		if t, ok := in.Answer[i].(*dns.A); ok {
			ips[i] = t.A.String()
		}
	}
	return ips, nil
}

func (r *Resolver) autoDeleteExpired() {
	rate := time.Second * 1
	timer := time.NewTimer(rate)
	for {
		select {
		case <-r.ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			r.deleteExpired()
			timer.Reset(rate)
		}
	}
}
