package executor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/millken/httpctl/config"
	"github.com/millken/httpctl/core"
	"github.com/millken/httpctl/log"
	"go.uber.org/zap"
)

type SourceMapExecutor struct {
	cfg     config.SourceMapExecutor
	log     *zap.Logger
	mu      sync.RWMutex
	ch      chan string
	process map[string]bool
}

func newSourceMapExecutor(ctx context.Context, cfg config.SourceMapExecutor) Executor {
	e := &SourceMapExecutor{
		cfg:     cfg,
		log:     log.Logger("sourcemap_executor"),
		process: make(map[string]bool),
		ch:      make(chan string),
	}
	go e.worker(ctx)
	return e
}

func (e *SourceMapExecutor) Writer(req *core.RequestHeader, resHeader *core.ResponseHeader) io.Writer {
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
	url := "http://"
	if req.GetHTTPS() {
		url = "https://"
	}
	url = fmt.Sprintf("%s%s%s", url, req.Host(), string(req.RequestURI()))

	if strings.HasSuffix(url, ".js") || strings.HasSuffix(url, ".css") {
		e.ch <- url
	}
	// dir, filename := filepath.Split(string(url))
	// if filename == "" {
	// 	filename = "index.html"
	// }
	// dir = fmt.Sprintf("%s/%s", e.cfg.OutputPath, dir)
	// dfile := fmt.Sprintf("%s%s", dir, filename)
	// if _, err := os.Stat(dfile); os.IsExist(err) {
	// 	e.log.Error("can not stat", zap.String("dfile", dfile), zap.Error(err))
	// 	return nil
	// }
	// stat, err := os.Stat(dir)
	// if os.IsNotExist(err) || !stat.IsDir() {
	// 	if err = os.MkdirAll(dir, 0644); err != nil {
	// 		e.log.Error("can not mkdir", zap.String("dir", dir), zap.Error(err))
	// 		return nil
	// 	}
	// }

	// fhandler, err := os.Create(dfile)
	// if err != nil {
	// 	e.log.Error("can not create file", zap.String("file", dfile), zap.Error(err))
	// 	return nil
	// }
	// e.log.Info("generate site file", zap.String("file", dfile))
	return nil
}

func (e *SourceMapExecutor) fetchSource(sourceURL string) (source []byte, err error) {
	var res *http.Response

	client := &http.Client{
		Transport: core.CreateHTTPTransport(nil),
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	res, err = client.Get(sourceURL + ".map")
	if err != nil {
		return
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("sourceMap URL request return != 200")
	}

	source, err = ioutil.ReadAll(res.Body)
	defer res.Body.Close()

	return
}

// writeFile writes content to file at path p.
func (e *SourceMapExecutor) writeFile(p string, content []byte) error {
	p = filepath.Clean(p)

	if _, err := os.Stat(filepath.Dir(p)); os.IsNotExist(err) {
		// Using MkdirAll here is tricky, because even if we fail, we might have
		// created some of the parent directories.
		err = os.MkdirAll(filepath.Dir(p), 0700)
		if err != nil {
			return err
		}
	}

	return ioutil.WriteFile(p, content, 0600)
}
func (e *SourceMapExecutor) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case ch := <-e.ch:
			u, _ := url.Parse(ch)
			p := fmt.Sprintf("%s/%s/%s.map", e.cfg.OutputPath, strings.ReplaceAll(u.Host, ":", "_"), u.Path)
			p = filepath.Clean(p)
			if _, err := os.Stat(p); err == nil {
				continue
			}
			go func() {
				source, err := e.fetchSource(ch)
				if err != nil {
					e.log.Warn("failed to fetch sourcemap", zap.String("url", ch), zap.Error(err))
				}
				err = e.writeFile(p, source)
				if err != nil {
					e.log.Error("failed to write sourcemap", zap.String("url", ch), zap.String("path", p), zap.Error(err))
				}
				e.log.Info("successfuly to write sourcemap", zap.String("url", ch), zap.String("path", p))
			}()

		}
	}
}
