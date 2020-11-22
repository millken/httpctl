package proxy

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/millken/httpctl/core"
	"github.com/millken/httpctl/executor"
	"github.com/millken/httpctl/log"

	"github.com/millken/httpctl/resolver"
	"go.uber.org/zap"
)

type HttpProxy struct {
	execute    *executor.Execute
	resolver   *resolver.Resolver
	bufferPool *core.BufferPool
	log        *zap.Logger
}

func NewHttpProxy(resolver *resolver.Resolver, execute *executor.Execute) *HttpProxy {
	p := &HttpProxy{
		execute:    execute,
		resolver:   resolver,
		bufferPool: core.BufferPool4k,
		log:        log.Logger("http"),
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
		Transport: createTransport(nil),
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	response, err := client.Do(req)
	if err != nil {
		p.log.Error("client do request", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer response.Body.Close()
	for k, v := range response.Header {
		if len(v) < 2 {
			w.Header().Set(k, v[0])
		} else {
			w.Header().Set(k, strings.Join(v, ""))
		}
	}

	buffer = p.bufferPool.Get()
	writer = io.MultiWriter(w, buffer)

	_, _ = io.Copy(writer, response.Body)
	var reader io.Reader
	switch response.Header.Get("Content-Encoding") {
	case "br":
		reader = brotli.NewReader(buffer)
	case "gzip":
		reader, err = gzip.NewReader(buffer)
		if err != nil {
			p.log.Error("gzip.NewReader", zap.Error(err))
		}
	default:
		reader = buffer
	}
	//io.Copy(os.Stdout, reader)
	reqHeader := &core.RequestHeader{}
	reqHeader.SetHost(req.Host)
	reqHeader.SetRequestURI(req.URL.RequestURI())
	reqHeader.SetMethod(req.Method)
	reqHeader.SetUserAgent(req.UserAgent())
	reqHeader.SetContentType(req.Header.Get("Content-Type"))
	if req.Header.Get("Connection") == "close" {
		reqHeader.SetConnectionClose()
	}
	if req.URL.Scheme == "https" {
		reqHeader.SetHTTPS()
	}
	resHeader := &core.ResponseHeader{}
	resHeader.SetContentType(response.Header.Get("Content-Type"))
	resHeader.SetStatusCode(response.StatusCode)

	copyWriter := io.MultiWriter(p.execute.Writer(reqHeader, resHeader)...)

	io.Copy(copyWriter, reader)
	p.bufferPool.Put(buffer)

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
	p.log.Debug("resolver request host", zap.String("host", req.Host), zap.Any("ip", ips))
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
