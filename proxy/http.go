package proxy

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/millken/httpctl/resolver"
)

type HttpProxy struct {
	resolver *resolver.Resolver
	buffer   *bytes.Buffer
}

func NewHttpProxy(resolver *resolver.Resolver) *HttpProxy {
	p := &HttpProxy{
		resolver: resolver,
		buffer:   BufferPool4k.Get(),
	}
	return p
}

func (p *HttpProxy) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	var writer io.Writer

	p.modifyRequest(req)
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	tres, err := client.Do(req)
	if err != nil {
		log.Printf("http client : %s", err)
		fmt.Fprintf(resp, "http client : %s", err)
		return
	}
	defer tres.Body.Close()
	for k, v := range tres.Header {
		if len(v) < 2 {
			resp.Header().Set(k, v[0])
		} else {
			resp.Header().Set(k, strings.Join(v, ""))
		}
	}
	if tres.StatusCode == 200 {
		writer = resp
	} else {
		writer = resp
	}
	_, _ = io.Copy(writer, tres.Body)
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

func (p *HttpProxy) modifyRequest(req *http.Request) {
	//log.Printf("req: %v", req)
	ips, err := p.resolver.Get(req.Host)
	if err != nil {
		log.Printf("domain %s resolver err: %s\n", req.Host, err)
	}
	if req.TLS == nil {
		req.URL.Scheme = "http"
	} else {
		req.URL.Scheme = "https"
	}
	//req.Header.Set("Accept-Encoding", "deflate")
	//req.Header.Set("Connection", "close")
	req.URL.Host = ips[0]
	req.RequestURI = ""
}

func (p *HttpProxy) ListenAndServe(addr string) error {

	return http.ListenAndServe(addr, p)
}

func (p *HttpProxy) ListenAndServeTLS(addr string, certFile string, keyFile string) error {

	return http.ListenAndServeTLS(addr, certFile, keyFile, p)
}
