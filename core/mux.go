package core

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/lucas-clemente/quic-go/http3"
	"github.com/millken/httpctl/resolver"
	"github.com/pkg/errors"
	"golang.org/x/net/http2"
)

type Mux struct {
	resolver *resolver.Resolver
	// The middleware stack
	middlewares []func(http.Handler) http.Handler
}

func NewMux(resolver *resolver.Resolver) *Mux {
	mux := &Mux{
		resolver: resolver,
	}
	return mux
}

// ServeHTTP is the single method of the http.Handler interface
func (mx *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	Chain(mx.middlewares...).Handler(mx.proxyHandler()).ServeHTTP(w, r)
}

// Use appends a middleware handler to the Mux middleware stack.
func (mx *Mux) Use(middlewares ...func(http.Handler) http.Handler) {
	mx.middlewares = append(mx.middlewares, middlewares...)
}

//curl https://babel-api.mainnet.iotex.io  -H "User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/102.0.0.0 Safari/537.36" -v --http2-prior-knowledge
func (mx *Mux) proxyHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Read the content
		var rBytes []byte
		var err error
		if r.Body != nil {
			rBytes, err = io.ReadAll(r.Body)
			if err != nil {
				log.Printf("Failed to read request body: %v", err)
				return
			}
			r.Body = io.NopCloser(bytes.NewBuffer(rBytes))
		}
		response, err := mx.handleHTTP(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		//r.Body has been read, so we can renew it.
		if r.Body != nil {
			r.Body = io.NopCloser(bytes.NewBuffer(rBytes))
		}

		w.WriteHeader(response.StatusCode)
		for name, values := range response.Header {
			w.Header()[name] = values
		}
		// bodyBuff, _ := io.ReadAll(response.Body)
		// w.Write(bodyBuff)
		defer response.Body.Close()
		if _, err := io.Copy(w, response.Body); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (mx *Mux) handleHTTP(r *http.Request) (*http.Response, error) {
	// ips, _, err := mx.resolver.Lookup(r.Host)
	// if err != nil {
	// 	return nil, err
	// }
	if r.TLS == nil {
		r.URL.Scheme = "http"
	} else {
		r.URL.Scheme = "https"
	}
	r.URL.Host = r.Host
	r.RequestURI = ""

	client := &http.Client{
		Transport: CreateHTTPTransport(nil),
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return client.Do(r)
}

func (mx *Mux) handleHTTP2(r *http.Request) (*http.Response, error) {
	if r.TLS == nil {
		return nil, fmt.Errorf("http2 only support https")
	}

	ips, _, err := mx.resolver.Lookup(r.Host)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to lookup host '%s'", r.Host)
	}
	r.URL.Scheme = "https"
	r.RequestURI = ""
	conf := &tls.Config{
		NextProtos: []string{"h2"},
		ServerName: r.Host,
	}
	host, port := ips[0], r.URL.Port()
	if port == "" {
		port = "443"
	}
	dialHost := net.JoinHostPort(host, port)

	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 60 * time.Second,
		DualStack: true,
	}
	tcpConn, err := tls.DialWithDialer(dialer, "tcp", dialHost, conf)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to dial host '%s'", dialHost)
	}
	defer tcpConn.Close()

	t := http2.Transport{}
	http2ClientConn, err := t.NewClientConn(tcpConn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to new client conn")
	}

	return http2ClientConn.RoundTrip(r)
}

func (mx *Mux) handleHTTP3(r *http.Request) (*http.Response, error) {
	if r.TLS == nil {
		return nil, fmt.Errorf("http2 only support https")
	}

	r.URL.Scheme = "https"
	r.URL.Host = r.Host
	r.RequestURI = ""

	var qconf quic.Config

	roundTripper := &http3.RoundTripper{
		Dial: func(ctx context.Context, network, addr string, tlsCfg *tls.Config, cfg *quic.Config) (quic.EarlyConnection, error) {
			host, port, _ := net.SplitHostPort(addr)
			ips, _, err := mx.resolver.Lookup(host)
			if err != nil {
				return nil, err
			}
			if port == "" {
				port = "443"
			}
			dialHost := net.JoinHostPort(ips[0], port)
			udpAddr, err := net.ResolveUDPAddr("udp", dialHost)
			if err != nil {
				return nil, err
			}
			udpConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
			if err != nil {
				return nil, err
			}

			return quic.DialEarly(udpConn, udpAddr, r.Host, tlsCfg, cfg)
		},
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		QuicConfig: &qconf,
	}
	defer roundTripper.Close()
	hclient := &http.Client{
		Transport: roundTripper,
	}

	return hclient.Do(r)
}
