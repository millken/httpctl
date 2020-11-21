package proxy

import "io"

type Context struct {
	RequestHeader  *RequestHeader
	ResponseHeader *ResponseHeader
	ResponseBody   io.Reader
}

type Proxy interface {
	ListenAndServe(addr string) error
	ListenAndServeTLS(addr string, certFile string, keyFile string) error
}
