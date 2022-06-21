package core

import (
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/proxy"
)

func CreateHTTPTransport(localAddr net.Addr) http.RoundTripper {
	proxyHost := os.Getenv("PROXY_HOST")
	proxyHost = "127.0.0.1:1080"
	log.Printf("proxyHost: %s", proxyHost)
	baseDialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}
	if localAddr != nil {
		baseDialer.LocalAddr = localAddr
	}

	tr := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           baseDialer.DialContext,
		DisableKeepAlives:     true,
		MaxIdleConns:          100,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConnsPerHost:   runtime.GOMAXPROCS(0) + 1,
		IdleConnTimeout:       7 * time.Second,
		// TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
	}

	if proxyHost != "" {
		dialSocksProxy, err := proxy.SOCKS5("tcp", proxyHost, nil, baseDialer)
		if err != nil {
			log.Println("Error creating SOCKS5 proxy, using HTTP_PROXY or direct connection")
		} else if contextDialer, ok := dialSocksProxy.(proxy.ContextDialer); ok {
			tr.DialContext = contextDialer.DialContext
		} else {
			log.Println("Failed type assertion to DialContext")
		}
		// logger.Debug("Using SOCKS5 proxy for http client",
		// 	zap.String("host", proxyHost),
		// )
		log.Println("Using SOCKS5 proxy for http client")
	}
	return tr
}

func CreateHTTP2Transport(localAddr net.Addr) http.RoundTripper {
	return &http2.Transport{
		AllowHTTP: true,
		DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
			return tls.Dial(network, addr, cfg)
		},
	}
}
