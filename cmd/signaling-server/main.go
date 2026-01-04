package main

import (
	"context"
	"log"
	"net"
	"p2p-msg/internal/stun"
	"sync"
)

func main() {

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		addr, err := net.ResolveUDPAddr("udp", "0.0.0.0:8484")
		if err != nil {
			log.Fatalln(err)
		}

		stunServer, err := stun.NewServer(addr)
		if err != nil {
			log.Fatalln(err)
		}
		defer stunServer.Dispose()

		stunServer.Serve(context.Background())
	}()
	log.Println("Http server started")

	wg.Wait()
	log.Println("All background processes are finished, shutting down...")
}
