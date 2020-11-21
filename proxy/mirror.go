package proxy

import (
	"github.com/millken/httpctl/log"
	"go.uber.org/zap"
)

type mirror struct {
	proxyCtx *Context
	log      *zap.Logger
}

func newMirror(proxyCtx *Context) *mirror {
	return &mirror{
		proxyCtx: proxyCtx,
		log:      log.Logger("mirror"),
	}
}
