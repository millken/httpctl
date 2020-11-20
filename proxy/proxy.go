package proxy

type Context struct {
}

type Proxy interface {
	ListenAndServe(addr string) error
	ListenAndServeTLS(addr string, certFile string, keyFile string) error
}
