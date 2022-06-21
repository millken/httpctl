package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/millken/httpctl/config"
	"github.com/millken/httpctl/core"
	"github.com/millken/httpctl/log"
	"github.com/millken/httpctl/middleware"
	"github.com/millken/httpctl/resolver"

	"github.com/millken/httpctl/certer"
	"go.uber.org/zap"
)

const version = "2.0.0"

const (
	ConfigPath = "HttpCtlConfigPath"
)

func main() {
	configPath := os.Getenv(ConfigPath)
	if configPath == "" {
		configPath = "config.yaml"
	}
	cfg, err := config.New(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to parse config: %v\n", err)
		os.Exit(1)
	}

	if err := log.InitLoggers(cfg.Log, cfg.SubLogs); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to init logger: %v\n", err)
		os.Exit(1)
	}
	log.L().Info("loading config", zap.Any("config", fmt.Sprintf("%+v", cfg)))

	// ctx := context.Background()
	// execute := executor.NewExecutor(ctx, cfg.Executor)

	// var proxyer proxy.Proxy
	resolvers := resolver.NewResolver(cfg.Server.Resolver)
	// proxyer = proxy.NewHttpProxy(resolvers, execute)

	mux := core.NewMux(resolvers)
	mux.Use(middleware.LoggingHandler(os.Stdout))
	mux.Use(middleware.HttpLogHandler)
	certCA := certer.NewCertCA()
	if err := certCA.LoadCA(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to init certificate: %v\n", err)
		os.Exit(1)
	}
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := http.ListenAndServe(cfg.Server.Http.Listen, mux); err != nil {
			log.L().Fatal("Failed to bind on the given interface (HTTP): ", zap.Error(err))
		}

	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		config := &tls.Config{
			GetCertificate: certCA.GetCertificate,
		}
		ln, err := tls.Listen("tcp", cfg.Server.Https.Listen, config)
		if err != nil {
			log.L().Fatal("listen error ", zap.Error(err))
		}
		defer ln.Close()
		if err := http.Serve(ln, mux); err != nil {
			log.L().Fatal("Failed to bind on the given interface (HTTPS): ", zap.Error(err))
		}
	}()

	wg.Wait()

}
