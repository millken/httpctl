package executor

import (
	"context"
	"io"

	"github.com/millken/httpctl/config"
	"github.com/millken/httpctl/core"
	"github.com/millken/httpctl/log"
	"go.uber.org/zap"
)

type FlowExecutor struct {
	cfg config.FlowExecutor
	log *zap.Logger
}

func newFlowExecutor(ctx context.Context, cfg config.FlowExecutor) Executor {
	return &FlowExecutor{
		cfg: cfg,
		log: log.Logger("flow_executor"),
	}
}

func (e *FlowExecutor) Writer(req *core.RequestHeader, resHeader *core.ResponseHeader) io.Writer {
	return nil
}
