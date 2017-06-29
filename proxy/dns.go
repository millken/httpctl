package proxy

import (
	"errors"
	"log"

	"github.com/miekg/dns"
	"github.com/millken/raphanus"
)

var dnscache = raphanus.New(8)

func (h *Handler) queryDNS(domain string) (string, error) {
	var value string
	if value, err := dnscache.GetStr(domain); err == nil {
		return value, err
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
	if t, ok := in.Answer[0].(*dns.A); ok {
		log.Printf("domain: %s => %s\n", domain, t.A.String())
		value = t.A.String()
		dnscache.SetStr(domain, value, 600)
	}
	return value, nil
}
