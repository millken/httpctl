package proxy

type HttpProxy struct {
}

func (p *HttpProxy) ListenAndServe(addr string) error {

	return nil
}

func (p *HttpProxy) ListenAndServeTLS(addr string, certFile string, keyFile string) error {

	return nil
}
