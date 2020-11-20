package proxy

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/millken/httpctl/log"
	"go.uber.org/zap"
)

type mirror struct {
	proxyCtx *proxyContext
	log      *zap.Logger
}

func newMirror(proxyCtx *proxyContext) *mirror {
	return &mirror{
		proxyCtx: proxyCtx,
		log:      log.Logger("mirror"),
	}
}

func (m *mirror) Work() {
	r := m.proxyCtx.Request
L:
//check content-encoding
	for _, enc := range strings.Split(r.Header.Get("Accept-Encoding"), ",") {
		switch strings.TrimSpace(enc) {
		case "gzip":
			zr, err := gzip.NewReader(m.proxyCtx.Buffer)
			if err != nil {
				m.log.Error("gzip.NewReader", zap.Error(err))
				fmt.Printf("%s", m.proxyCtx.Buffer.Bytes())
				break L
			}

			if _, err := io.Copy(os.Stdout, zr); err != nil {
				m.log.Error("io copy", zap.Error(err))
			}

			if err := zr.Close(); err != nil {
				m.log.Error("gzip.Close", zap.Error(err))
			}
			break L
		case "deflate":
			io.Copy(os.Stdout, m.proxyCtx.Buffer)
			break L
		}
	}
}
