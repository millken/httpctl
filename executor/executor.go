package executor

import (
	"context"
	"io"
	"io/ioutil"

	"github.com/millken/httpctl/config"
	"github.com/millken/httpctl/core"
	"github.com/millken/httpctl/log"
	"go.uber.org/zap"
)

type Executor interface {
	Writer(*core.RequestHeader, *core.ResponseHeader) io.Writer
}

type Execute struct {
	cfg       config.Executor
	log       *zap.Logger
	executors []Executor
}

func NewExecutor(ctx context.Context, cfg config.Executor) *Execute {
	e := &Execute{
		cfg:       cfg,
		log:       log.Logger("executor"),
		executors: []Executor{},
	}
	if cfg.Example.Enable {
		e.executors = append(e.executors, newExampleExecutor(ctx, cfg.Example))
	}
	if cfg.SiteCopy.Enable {
		e.executors = append(e.executors, newSiteCopyExecutor(ctx, cfg.SiteCopy))
	}
	if cfg.SourceMap.Enable {
		e.executors = append(e.executors, newSourceMapExecutor(ctx, cfg.SourceMap))
	}
	return e
}

func (e *Execute) Writer(req *core.RequestHeader, res *core.ResponseHeader) []io.Writer {
	writers := []io.Writer{ioutil.Discard}
	for _, executor := range e.executors {
		if writer := executor.Writer(req, res); writer != nil {
			writers = append(writers, writer)
		}
	}
	return writers
}
