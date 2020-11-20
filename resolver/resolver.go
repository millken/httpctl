package resolver

import (
	"context"
	"errors"
	"fmt"
	"net"
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
	resolver string
	cache    map[string]Item
}

func NewResolver(resolver string) *Resolver {
	r := &Resolver{
		resolver: resolver,
		cache:    make(map[string]Item),
	}
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

	return r.lookupHost(host)
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

func (r *Resolver) lookupHost(host string) ([]string, error) {
	m1 := new(dns.Msg)
	m1.Id = dns.Id()
	m1.RecursionDesired = true
	m1.Question = make([]dns.Question, 1)
	m1.Question[0] = dns.Question{dns.Fqdn(host), dns.TypeA, dns.ClassINET}

	c := new(dns.Client)
	ctx, _ := context.WithTimeout(context.Background(), ResolverTimeout)
	in, _, err := c.ExchangeContext(ctx, m1, r.resolver+":53")

	if err != nil {
		return nil, err
	}

	l := len(in.Answer)
	if l == 0 {
		return nil, errors.New(" answer has empty")
	}
	ips := []string{}
	for i := 0; i < l; i++ {
		switch in.Answer[i].(type) {
		case *dns.A:
			a := in.Answer[i].(*dns.A)
			ips = append(ips, a.A.String())
		case *dns.CNAME:
			cname := in.Answer[i].(*dns.CNAME)
			ipa, _ := net.LookupIP(cname.Target)
			for _, ip := range ipa {
				if ipv4 := ip.To4(); ipv4 != nil {
					ips = append(ips, ipv4.String())
				}
			}
		}
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("in.Answer : %v", in.Answer)
	}
	r.Lock()
	r.cache[host] = Item{ips, time.Now().Add(DefaultExpiration).UnixNano()}
	r.Unlock()
	return ips, nil
}
