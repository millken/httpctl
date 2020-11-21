package executor

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/millken/httpctl/config"
	"github.com/millken/httpctl/core"
	"go.uber.org/zap"
)

type ExampleExecutor struct {
	log     *zap.Logger
	ctxChan chan *core.Context
}

func (e *ExampleExecutor) Handler(ctx *core.Context) {
	e.ctxChan <- ctx
}

func (e *ExampleExecutor) Run(ctx context.Context, cfg config.Executor) {
	if !cfg.Example.Enable {
		return
	}
	for {
		select {
		case c, ok := <-e.ctxChan:
			if !ok {
				return
			}
			e.log.Debug(fmt.Sprintf("%s %s", c.RequestHeader.Host(), c.RequestHeader.RequestURI()))
			io.Copy(ioutil.Discard, c.ResponseBody)
		case <-ctx.Done():
			return
		}
	}
}

func init() {
	RegisterExecutor("example_executor", func(log *zap.Logger) interface{} {
		return &ExampleExecutor{
			log:     log,
			ctxChan: make(chan *core.Context, 100),
		}
	})
}
