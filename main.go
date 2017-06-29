package main

import (
	"log"
	"net/http"
	"sync"

	"github.com/millken/downloader/proxy"
)

func main() {
	handler := proxy.NewHandler("192.168.4.1")

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := http.ListenAndServe("127.0.0.1:80", handler); err != nil {
			log.Fatalf("Failed to bind on the given interface (HTTP): ", err)
		}

	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := http.ListenAndServeTLS("127.0.0.1:443", "./ssl/cert.pem", "./ssl/key.pem", handler); err != nil {
			log.Fatalf("Failed to bind on the given interface (HTTPS): ", err)
		}
	}()

	wg.Wait()

}
