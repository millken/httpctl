package proxy

import (
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"runtime"
	"time"
)

type Context struct {
	RequestHeader  *RequestHeader
	ResponseHeader *ResponseHeader
	ResponseBody   io.Reader
}

type Proxy interface {
	ListenAndServe(addr string) error
	ListenAndServeTLS(addr string, certFile string, keyFile string) error
}

func createTransport(localAddr net.Addr) *http.Transport {
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
