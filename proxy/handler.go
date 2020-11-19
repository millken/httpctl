package proxy

import (
	"bytes"
	"compress/gzip"
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
	buf      *bytes.Buffer
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
	var buffer *bytes.Buffer
	buffer = nil
	if tres.StatusCode == 200 {
		buffer = BufferPool4k.Get()
		h.buf = buffer
		writer = h.archiver(resp, req)
	} else {
		writer = resp
	}
	_, _ = io.Copy(writer, tres.Body)
	if buffer != nil {
		zr, err := gzip.NewReader(buffer)
		if err != nil {
			log.Fatal(err)
		}

		if _, err := io.Copy(os.Stdout, zr); err != nil {
			log.Fatal(err)
		}

		if err := zr.Close(); err != nil {
			log.Fatal(err)
		}
		//log.Printf("\n%s\n", buffer.Bytes())
	}
}

func (h *Handler) archiver(resp http.ResponseWriter, req *http.Request) io.Writer {
	var writer io.Writer
	url := fmt.Sprintf("%s%s", req.Host, req.URL.Path)
	dir, filename := filepath.Split(url)
	if filename == "" {
		filename = "index.html"
	}
	dir = fmt.Sprintf("archives/%s", dir)
	dfile := fmt.Sprintf("%s%s", dir, filename)
	if _, err := os.Stat(dfile); os.IsExist(err) {
		return resp
	}
	stat, err := os.Stat(dir)
	if os.IsNotExist(err) || !stat.IsDir() {
		if err = os.MkdirAll(dir, 0644); err != nil {
			log.Printf("can not mkdir : %s, %s\n", dir, err)
			return resp
		}

	}

	//fhandler, _ := os.Create(dfile)
	//defer fhandler.Close()
	//log.Printf("url: %s, dir: %s, filename: %s\n", url, dir, filename)
	writer = io.MultiWriter(resp, h.buf)
	return writer
}

func (h *Handler) modifyRequest(req *http.Request) {
	log.Printf("req: %v", req)
	aRecord, err := h.queryDNS(req.Host)
	if err != nil {
		log.Printf("queryDNS err: %s\n", err)
	}
	if req.TLS == nil {
		req.URL.Scheme = "http"
	} else {
		req.URL.Scheme = "https"
	}
	//req.Header.Set("Accept-Encoding", "deflate")
	//req.Header.Set("Connection", "close")
	req.URL.Host = aRecord
	req.RequestURI = ""
}

func NewHandler(ns string) http.Handler {
	return &Handler{resolver: ns}
}
