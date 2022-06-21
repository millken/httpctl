package middleware

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os"
	"sort"
	"strconv"
	"strings"
)

func HttpLogHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := httptest.NewRecorder()
		next.ServeHTTP(rec, r)
		for k, v := range rec.Header() {
			w.Header()[k] = v
		}

		w.WriteHeader(rec.Code)

		b := rec.Body.Bytes()
		w.Write(b)
		dump, err := httputil.DumpRequest(r, true)
		if err != nil {
			log.Println(err)
			return
		}
		fmt.Fprintf(os.Stdout, "REQUEST \n%s\n", dump)

		//todo: write response body to file, noheaders because it's a http2.0 response
		dump, err = dumpResponse(rec)
		if err != nil {
			log.Println(err)
			return
		}
		fmt.Fprintf(os.Stdout, "RESPONSE \n%s\n", dump)
	})
}

func dumpResponse(rr *httptest.ResponseRecorder) ([]byte, error) {
	var b bytes.Buffer
	r := rr.Result()
	// Status line
	text := r.Status
	if text == "" {
		text = http.StatusText(r.StatusCode)
	} else {
		// Just to reduce stutter, if user set r.Status to "200 OK" and StatusCode to 200.
		// Not important.
		text = strings.TrimPrefix(text, strconv.Itoa(r.StatusCode)+" ")
	}

	if _, err := fmt.Fprintf(&b, "HTTP/%d.%d %03d %s\r\n", r.ProtoMajor, r.ProtoMinor, r.StatusCode, text); err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(rr.Header()))
	for k := range rr.Header() {
		keys = append(keys, k)
	}
	if len(keys) > 0 {
		sort.Strings(keys)
	}

	for _, k := range keys {
		if _, err := fmt.Fprintf(&b, "%s: %s\r\n", k, strings.Join(rr.Header()[k], ",")); err != nil {
			return nil, err
		}
	}
	// End-of-header
	if _, err := io.WriteString(&b, "\r\n"); err != nil {
		return nil, err
	}
	// Body
	var b1 bytes.Buffer
	b1.Reset()
	r.Body, _ = drainBody(r.Body)
	r.Write(&b1)
	bb := b1.Bytes()
	n := bytes.Index(bb, []byte("\r\n\r\n"))
	if n > -1 {
		bb = bb[n+4:]
	}

	b.Write(bb)

	return b.Bytes(), nil
}

func drainBody(b io.ReadCloser) (r2 io.ReadCloser, err error) {
	if b == nil || b == http.NoBody {
		// No copying needed. Preserve the magic sentinel meaning of NoBody.
		return http.NoBody, nil
	}
	var buf bytes.Buffer
	if _, err = buf.ReadFrom(b); err != nil {
		return b, err
	}
	if err = b.Close(); err != nil {
		return b, err
	}
	return io.NopCloser(bytes.NewReader(buf.Bytes())), nil
}
