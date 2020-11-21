package executor

import (
	"context"

	"github.com/millken/httpctl/config"
	"github.com/millken/httpctl/core"
	"github.com/millken/httpctl/log"
	"go.uber.org/zap"
)

var executors = make(map[string]func(*zap.Logger) interface{})

func RegisterExecutor(name string, executor func(*zap.Logger) interface{}) {
	if executor == nil {
		return
	}

	if _, ok := executors[name]; ok {
		log.L().Fatal("Register called twice for filter ", zap.String("name", name))
	}

	executors[name] = executor
}

type Executor interface {
	Handler(ctx *core.Context)
	Run(ctx context.Context, cfg config.Executor)
}

type Execute struct {
	cfg       config.Executor
	log       *zap.Logger
	executors map[string]Executor
}

func NewExecutor(cfg config.Executor) *Execute {
	return &Execute{
		cfg:       cfg,
		log:       log.Logger("executor"),
		executors: make(map[string]Executor),
	}
}

func (e *Execute) Start(ctx context.Context) {
	e.log.Info("executor starting")
	for name, executor := range executors {
		exe := executor(e.log).(Executor)
		e.executors[name] = exe
		go exe.Run(ctx, e.cfg)
	}
}

func (e *Execute) Handler(ctx *core.Context) {
	for _, executor := range e.executors {
		executor.Handler(ctx)
	}
}
