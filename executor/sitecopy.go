package executor

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
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

func newSiteCopyExecutor(ctx context.Context, cfg config.SiteCopyExecutor) Executor {
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
	if !hit || req.IsPost() || resHeader.StatusCode() != 200 {
		e.log.Warn("not exist in hosts or http method = post or status = 200")
		return nil
	}
	currentUrl := string(append(req.Host(), string(req.RequestURI())...))
	u, _ := url.Parse(currentUrl)
	fixUrl := u.Host + u.Path
	dir, filename := filepath.Split(filepath.Clean(fixUrl))
	if filename == "" {
		filename = "index.html"
	}
	if u.RawQuery != "" && sort.SearchStrings([]string{"css", "ttf", "woff"}, filepath.Ext(filename)[1:]) == 3 {
		e.log.Warn("skip url with query, otherwise css", zap.String("rawQuery", u.RawQuery), zap.String("ext", filepath.Ext(filename)))
		return nil
	}

	dir = fmt.Sprintf("%s/%s", e.cfg.OutputPath, dir)
	dfile := fmt.Sprintf("%s%s", dir, filename)
	if stat, err := os.Stat(dfile); !os.IsNotExist(err) && stat.Size() != 0 {
		e.log.Debug("skip existed file", zap.String("file", dfile), zap.Int64("size", stat.Size()))
		return nil
	}
	stat, err := os.Stat(dir)
	if os.IsNotExist(err) || !stat.IsDir() {
		if err = os.MkdirAll(dir, 0700); err != nil {
			e.log.Error("can not mkdir", zap.String("dir", dir), zap.Error(err))
			return nil
		}
	}

	fhandler, err := os.Create(dfile)
	if err != nil {
		e.log.Error("can not create file", zap.String("file", dfile), zap.Error(err))
		return nil
	}
	e.log.Info("generate site file", zap.String("file", dfile))
	return fhandler
}
