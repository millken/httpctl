package core

import (
	"crypto/tls"
	"net"
	"net/http"
	"runtime"
	"time"
)

func CreateHTTPTransport(localAddr net.Addr) *http.Transport {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}
	if localAddr != nil {
		dialer.LocalAddr = localAddr
	}
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       7 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConnsPerHost:   runtime.GOMAXPROCS(0) + 1,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
	}
}
