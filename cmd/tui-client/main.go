package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"p2p-msg/internal/config"
	"p2p-msg/internal/stun"
)

// TODO: replace with cmd args
const CLIENT_CONFIG_PATH = "cfg/client.json"

func main() {
	// read configs
	log.Printf("Loading JSON config from `%s`...", CLIENT_CONFIG_PATH)

	config, err := config.LoadFromJson(CLIENT_CONFIG_PATH)
	if err != nil {
		log.Fatalln("Can't load json config: ", err) // exit
	}

	log.Println("Config loaded")

	fullSignalingUrl := fmt.Sprintf("%s:%s", config.SignalingUrl, config.SignalingPort)

	clientAddr, err := net.ResolveUDPAddr("udp", "0.0.0.0:0")
	if err != nil {
		panic(err)
	}

	serverAddr, err := net.ResolveUDPAddr("udp", fullSignalingUrl) // Change IP to server's IP
	if err != nil {
		panic(err)
	}

	client, err := stun.NewClient(clientAddr, serverAddr, "tui", make([]byte, 0), make([]byte, 0))
	if err != nil {
		panic(err)
	}

	defer client.Dispose()

	ctx := context.Background()

	log.Println("All set up!")

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()

		client.Serve(ctx)
	}()

	// standby
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			time.Sleep(time.Duration(5) * time.Second)

			// Resolve the destination address (server address)
			dR, err := client.Discover()
			if err != nil {
				log.Println("Discover request failed with error: ", err)

				continue
			} else {
				log.Println("Got discover response: ", dR.String())
			}

			bR, err := client.Bind("test")
			if err != nil {
				log.Println("Bind request failed with error: ", err)

				continue
			} else {
				log.Println("Got bind response: ", bR.String())
			}

			rec, err := net.ResolveUDPAddr("udp", bR.Address)
			if err != nil {
				log.Println("Echo request error: ", err)

				continue
			}

			err = client.SendEcho(rec)
			if err != nil {
				log.Println("Bind request failed with error: ", err)

				continue
			} else {
				log.Println("SENT ECHO: ", rec)
			}

		}
	}()

	wg.Wait()
}
