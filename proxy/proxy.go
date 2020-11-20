package proxy

import (
	"bytes"
	"net/http"
)

type proxyContext struct {
	Request *http.Request
	Buffer  *bytes.Buffer
}

type Proxy interface {
	ListenAndServe(addr string) error
	ListenAndServeTLS(addr string, certFile string, keyFile string) error
}
