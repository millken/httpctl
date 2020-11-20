package proxy

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/millken/httpctl/log"

	"github.com/millken/httpctl/resolver"
	"go.uber.org/zap"
)

type HttpProxy struct {
	resolver *resolver.Resolver
	buffer   *bytes.Buffer
	log      *zap.Logger
}

func NewHttpProxy(resolver *resolver.Resolver) *HttpProxy {
	p := &HttpProxy{
		resolver: resolver,
		buffer:   BufferPool4k.Get(),
		log:      log.Logger("http"),
	}
	return p
}

func (p *HttpProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var writer io.Writer
	var buffer *bytes.Buffer
	req, err := p.modifyRequest(r)
	if err != nil {
		p.log.Error("modify request", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	tres, err := client.Do(req)
	if err != nil {
		p.log.Error("client do request", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tres.Body.Close()
	for k, v := range tres.Header {
		if len(v) < 2 {
			w.Header().Set(k, v[0])
		} else {
			w.Header().Set(k, strings.Join(v, ""))
		}
	}
	buffer = BufferPool4k.Get()
	writer = io.MultiWriter(w, buffer)

	proxyCtx := &proxyContext{
		Request: req,
		Buffer:  buffer,
	}
	mirr := newMirror(proxyCtx)

	_, _ = io.Copy(writer, tres.Body)
	mirr.Work()
	BufferPool4k.Put(buffer)
	// if p.buffer != nil {
	// 	zr, err := gzip.NewReader(p.buffer)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}

	// 	if _, err := io.Copy(os.Stdout, zr); err != nil {
	// 		log.Fatal(err)
	// 	}

	// 	if err := zr.Close(); err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	//log.Printf("\n%s\n", buffer.Bytes())
	// }
}

func (p *HttpProxy) modifyRequest(r *http.Request) (*http.Request, error) {
	req := r.Clone(context.Background())
	ips, err := p.resolver.Get(req.Host)
	if err != nil {
		return nil, fmt.Errorf("domain %s resolver err: %s", req.Host, err)
	}
	if req.TLS == nil {
		req.URL.Scheme = "http"
	} else {
		req.URL.Scheme = "https"
	}
	//req.Header.Set("Accept-Encoding", "deflate")
	//req.Header.Set("Connection", "close")
	p.log.Info("resolver get request host", zap.String("host", req.Host), zap.Any("ip", ips))
	req.URL.Host = ips[0]
	req.RequestURI = ""
	return req, nil
}

func (p *HttpProxy) ListenAndServe(addr string) error {

	return http.ListenAndServe(addr, p)
}

func (p *HttpProxy) ListenAndServeTLS(addr string, certFile string, keyFile string) error {

	return http.ListenAndServeTLS(addr, certFile, keyFile, p)
}
