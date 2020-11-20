package main

import (
	"flag"
	"fmt"
	"log"
	"sync"

	"github.com/millken/httpctl/proxy"
	"github.com/millken/httpctl/resolver"
)

const version = "2.0.0"

const (
	defaultResolver = `114.114.114.114`
	defaultAddress  = `127.0.0.1`
	defaultPort     = uint(80)
	defaultSSLPort  = uint(443)
)

var (
	flagAddress     = flag.String("l", defaultAddress, "Bind address.")
	flagResolver    = flag.String("r", defaultResolver, "Resolver address.")
	flagPort        = flag.Uint("p", defaultPort, "Port to bind to.")
	flagSSLPort     = flag.Uint("s", defaultSSLPort, "Port to bind to (SSL mode).")
	flagSSLCertFile = flag.String("c", "./ssl/cert.pem", "Path to root CA certificate.")
	flagSSLKeyFile  = flag.String("k", "./ssl/key.pem", "Path to root CA key.")
)

func main() {
	log.Printf("httpctl v%s // by millken\n", version)
	flag.Parse()

	var proxyer proxy.Proxy
	resolvers := resolver.NewResolver(*flagResolver)
	proxyer = proxy.NewHttpProxy(resolvers)
	//handler := proxy.NewHandler(*flagResolver)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := proxyer.ListenAndServe(fmt.Sprintf("%s:%d", *flagAddress, *flagPort)); err != nil {
			log.Fatalf("Failed to bind on the given interface (HTTP): ", err)
		}

	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := proxyer.ListenAndServeTLS(fmt.Sprintf("%s:%d", *flagAddress, *flagSSLPort), *flagSSLCertFile, *flagSSLKeyFile); err != nil {
			log.Fatalf("Failed to bind on the given interface (HTTPS): ", err)
		}
	}()

	wg.Wait()

}
