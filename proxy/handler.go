package proxy

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Handler struct {
	resolver string
}

func (h *Handler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	var writer io.Writer

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
	if tres.StatusCode == 200 {
		writer = h.archiver(resp, req)
	} else {
		writer = resp
	}
	io.Copy(writer, tres.Body)
}

func (h *Handler) archiver(resp http.ResponseWriter, req *http.Request) io.Writer {
	var writer io.Writer
	url := fmt.Sprintf("%s%s", req.Host, req.URL.Path)
	dir, filename := filepath.Split(url)
	if filename == "" {
		filename = "index.html"
	}
	stat, err := os.Stat(dir)
	if os.IsNotExist(err) || !stat.IsDir() {
		if err = os.MkdirAll(dir, 0644); err != nil {
			log.Printf("can not mkdir : %s, %s\n", dir, err)
			return resp
		}

	}

	fhandler, _ := os.Create(fmt.Sprintf("%s%s", dir, filename))
	log.Printf("url: %s, dir: %s, filename: %s\n", url, dir, filename)
	writer = io.MultiWriter(resp, fhandler)
	return writer
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
	req.Header.Set("Accept-Encoding", "deflate")
	req.URL.Host = aRecord
	req.RequestURI = ""
}

func NewHandler(ns string) http.Handler {
	return &Handler{resolver: ns}
}
