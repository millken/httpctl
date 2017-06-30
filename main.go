package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/millken/downloader/proxy"
)

const version = "1.0.0"

const (
	defaultResolver = `192.168.1.1`
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
	log.Printf("Downlader v%s // by millken\n", version)
	flag.Parse()

	handler := proxy.NewHandler(*flagResolver)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := http.ListenAndServe(fmt.Sprintf("%s:%d", *flagAddress, *flagPort), handler); err != nil {
			log.Fatalf("Failed to bind on the given interface (HTTP): ", err)
		}

	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := http.ListenAndServeTLS(fmt.Sprintf("%s:%d", *flagAddress, *flagSSLPort), *flagSSLCertFile, *flagSSLKeyFile, handler); err != nil {
			log.Fatalf("Failed to bind on the given interface (HTTPS): ", err)
		}
	}()

	wg.Wait()

}
