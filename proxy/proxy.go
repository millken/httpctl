package proxy

import (
	"bytes"
	"io/ioutil"
	"net/http"
)

type Proxy interface {
	ListenAndServe(addr string) error
	ListenAndServeTLS(addr string, certFile string, keyFile string) error
}

// DuplicateRequest duplicate http request
func DuplicateRequest(request *http.Request) (dup *http.Request) {
	var bodyBytes []byte
	if request.Body != nil {
		bodyBytes, _ = ioutil.ReadAll(request.Body)
	}
	request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
	dup = &http.Request{
		Method:        request.Method,
		URL:           request.URL,
		Proto:         request.Proto,
		ProtoMajor:    request.ProtoMajor,
		ProtoMinor:    request.ProtoMinor,
		Header:        request.Header,
		Body:          ioutil.NopCloser(bytes.NewBuffer(bodyBytes)),
		Host:          request.Host,
		ContentLength: request.ContentLength,
		Close:         true,
	}
	return
}
