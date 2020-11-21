package main

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/millken/httpctl/config"
	"github.com/millken/httpctl/executor"
	"github.com/millken/httpctl/log"

	"github.com/millken/httpctl/proxy"
	"github.com/millken/httpctl/resolver"
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

	ctx := context.Background()
	execute := executor.NewExecutor(cfg.Executor)
	execute.Start(ctx)

	var proxyer proxy.Proxy
	resolvers := resolver.NewResolver(cfg.Server.Resolver)
	proxyer = proxy.NewHttpProxy(resolvers, execute)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := proxyer.ListenAndServe(cfg.Server.Http.Listen); err != nil {
			log.L().Fatal("Failed to bind on the given interface (HTTP): ", zap.Error(err))
		}

	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := proxyer.ListenAndServeTLS(cfg.Server.Https.Listen, cfg.Server.Https.CertFile, cfg.Server.Https.KeyFile); err != nil {
			log.L().Fatal("Failed to bind on the given interface (HTTPS): ", zap.Error(err))
		}
	}()

	wg.Wait()

}
