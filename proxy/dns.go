package proxy

import (
	"errors"
	"log"

	"github.com/miekg/dns"
	"github.com/millken/gocache"
)


func (h *Handler) queryDNS(domain string) (string, error) {
	var val string
	value, ok := gocache.Get(domain)
	if ok {
		return value.(string), nil
	}
	m1 := new(dns.Msg)
	m1.Id = dns.Id()
	m1.RecursionDesired = true
	m1.Question = make([]dns.Question, 1)
	m1.Question[0] = dns.Question{dns.Fqdn(domain), dns.TypeA, dns.ClassINET}

	c := new(dns.Client)
	in, _, err := c.Exchange(m1, h.resolver+":53")

	if err != nil {
		return "", err
	}

	if len(in.Answer) == 0 {
		return "", errors.New(" answer has empty")
	}
	if t, ok := in.Answer[len(in.Answer)-1].(*dns.A); ok {
		log.Printf("domain: %s => %s\n", domain, t.A.String())
		val = t.A.String()
		gocache.Set(domain, val, 600)
	}
	return val, nil
}
