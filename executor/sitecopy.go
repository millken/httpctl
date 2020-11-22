package executor

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/millken/httpctl/config"
	"github.com/millken/httpctl/core"
	"github.com/millken/httpctl/log"
	"go.uber.org/zap"
)

type SiteCopyExecutor struct {
	cfg config.SiteCopyExecutor
	log *zap.Logger
}

func newSiteCopyExecutor(cfg config.SiteCopyExecutor) Executor {
	return &SiteCopyExecutor{
		cfg: cfg,
		log: log.Logger("sitecopy_executor"),
	}
}

func (e *SiteCopyExecutor) Writer(req *core.RequestHeader, resHeader *core.ResponseHeader) io.Writer {
	hit := false
	for _, host := range e.cfg.Hosts {
		if host == strings.ToLower(string(req.Host())) {
			hit = true
			break
		}
	}
	if !hit || req.IsPost() || bytes.Contains(req.RequestURI(), []byte("?")) || resHeader.StatusCode() != 200 {
		return nil
	}
	url := append(req.Host(), string(req.RequestURI())...)
	fmt.Printf("host = %s, uri = %s, method=%s", req.Host(), url, req.Method())
	dir, filename := filepath.Split(string(url))
	if filename == "" {
		filename = "index.html"
	}
	dir = fmt.Sprintf("%s/%s", e.cfg.OutputPath, dir)
	dfile := fmt.Sprintf("%s%s", dir, filename)
	if _, err := os.Stat(dfile); os.IsExist(err) {
		e.log.Error("can not stat", zap.String("dfile", dfile), zap.Error(err))
		return nil
	}
	stat, err := os.Stat(dir)
	if os.IsNotExist(err) || !stat.IsDir() {
		if err = os.MkdirAll(dir, 0644); err != nil {
			e.log.Error("can not mkdir", zap.String("dir", dir), zap.Error(err))
			return nil
		}
	}

	fhandler, err := os.Create(dfile)
	if err != nil {
		e.log.Error("can not create file", zap.String("file", dfile), zap.Error(err))
		return nil
	}
	return fhandler
}
