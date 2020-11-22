package executor

import (
	"io"
	"os"

	"github.com/millken/httpctl/config"
	"github.com/millken/httpctl/core"
	"github.com/millken/httpctl/log"
	"go.uber.org/zap"
)

type ExampleExecutor struct {
	cfg config.ExampleExecutor
	log *zap.Logger
}

func newExampleExecutor(cfg config.ExampleExecutor) Executor {
	return &ExampleExecutor{
		cfg: cfg,
		log: log.Logger("example_executor"),
	}
}

func (e *ExampleExecutor) Writer(req *core.RequestHeader, resHeader *core.ResponseHeader) io.Writer {
	return os.Stdout
}

// 	for {
// 		select {
// 		case c, ok := <-e.ctxChan:
// 			if !ok {
// 				return
// 			}
// 			var buf bytes.Buffer
// 			reader := io.TeeReader(c.ResponseBody, &buf)
// 			e.log.Debug(fmt.Sprintf("%s %s", c.RequestHeader.Host(), c.RequestHeader.RequestURI()))
// 			io.Copy(os.Stdout, reader)
// 		case <-ctx.Done():
// 			return
// 		}
// 	}
// }

// func init() {
// 	RegisterExecutor("example_executor", func(log *zap.Logger, cfg config.Executor) interface{} {
// 		return &ExampleExecutor{
// 			cfg:     cfg,
// 			log:     log,
// 		}
// 	})
// }
