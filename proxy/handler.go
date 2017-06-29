package proxy

import (
	"crypto/tls"
	"io"
	"log"
	"net/http"
	"strings"
)

type Handler struct {
	resolver string
}

func (h *Handler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	h.getFromOrigin(resp, req)
}

func (h *Handler) modifyRequest(req *http.Request) {
	aRecord, err := h.queryDNS(req.Host)
	if err != nil {
		log.Printf("queryDNS err: %s\n", err)
	}
	if req.TLS == nil {
		req.URL.Scheme = "http"
	} else {
		req.URL.Scheme = "https"
	}
	req.URL.Host = aRecord
	req.RequestURI = ""
}

func (h *Handler) getFromOrigin(resp http.ResponseWriter, req *http.Request) {
	h.modifyRequest(req)
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	tres, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	for k, v := range tres.Header {
		if len(v) < 2 {
			resp.Header().Set(k, v[0])
		} else {
			resp.Header().Set(k, strings.Join(v, ""))
		}
	}
	defer tres.Body.Close()
	io.Copy(resp, tres.Body)
}

func NewHandler(ns string) http.Handler {
	return &Handler{resolver: ns}
}
