package resolver

import (
	"context"
	"errors"
	"math/rand"
	"net"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

//https://github.com/parrotgeek1/ProxyDNS

var (
	DefaultNameServers               = []string{"8.8.8.8:53", "1.1.1.1:53"}
	DefaultExpiration  time.Duration = time.Minute * 10
	ResolverTimeout    time.Duration = time.Second * 7
)
var (
	ErrAnswerEmpty = errors.New("answer is empty")
	ErrIpEmpty     = errors.New("ip is empty")
)

type Item struct {
	Nameserver string
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
	janitor     *janitor
	nameservers []string
	cache       map[string]Item
}

func NewResolver(nameservers ...string) *Resolver {
	if len(nameservers) == 0 {
		nameservers = DefaultNameServers
	}
	r := &Resolver{
		nameservers: nameservers,
		cache:       make(map[string]Item),
	}
	runJanitor(r, DefaultExpiration/2)
	runtime.SetFinalizer(r, stopJanitor)
	return r
}

func (r *Resolver) Lookup(host string, nameservers ...string) ([]string, string, error) {
	if host == "" {
		return nil, "", errors.New("resolve host is empty")
	}
	r.RLock()
	item, found := r.cache[host]
	if found && !item.Expired() {
		r.RUnlock()
		return item.Object, item.Nameserver, nil
	}
	r.RUnlock()

	return r.lookupHost(host, nameservers...)
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

func (r *Resolver) lookupHost(host string, nameservers ...string) ([]string, string, error) {
	idx := strings.IndexRune(host, ':')
	if idx > -1 {
		host = host[:idx]
	}
	m1 := new(dns.Msg)
	m1.Id = dns.Id()
	m1.RecursionDesired = true
	m1.Question = make([]dns.Question, 1)
	m1.Question[0] = dns.Question{
		Name:   dns.Fqdn(host),
		Qtype:  dns.TypeA,
		Qclass: dns.ClassINET,
	}

	c := new(dns.Client)
	ctx, cancel := context.WithTimeout(context.Background(), ResolverTimeout)
	defer cancel()
	if len(nameservers) == 0 {
		nameservers = r.nameservers
	}
	nameserver := nameservers[rand.Intn(len(nameservers))]
	if !strings.Contains(nameserver, ":") {
		nameserver += ":53"
	}
	in, _, err := c.ExchangeContext(ctx, m1, nameserver)

	if err != nil {
		return nil, nameserver, err
	}

	l := len(in.Answer)
	if l == 0 {
		return nil, nameserver, ErrAnswerEmpty
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
		return nil, nameserver, ErrIpEmpty
	}
	r.Lock()
	r.cache[host] = Item{
		Nameserver: nameserver,
		Object:     ips,
		Expiration: time.Now().Add(DefaultExpiration).UnixNano(),
	}
	r.Unlock()
	return ips, nameserver, nil
}
