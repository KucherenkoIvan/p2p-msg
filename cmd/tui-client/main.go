package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"p2p-msg/internal/config"
	"p2p-msg/internal/datagram"
	"p2p-msg/internal/signaling"
	"strings"
	"sync"
	"syscall"
	"time"
)

// TODO: replace with cmd args
const CLIENT_CONFIG_PATH = "cfg/local-client.json"

func main() {
	exitchan := make(chan os.Signal, 1)
	signal.Notify(exitchan, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGKILL, syscall.SIGTERM)

	// read configs
	log.Printf("Loading JSON config from `%s`...", CLIENT_CONFIG_PATH)

	config, err := config.LoadFromJson(CLIENT_CONFIG_PATH)
	if err != nil {
		log.Fatalln("Can't load json config: ", err) // exit
	}

	log.Println("Config loaded")

	// resolve server
	fullSignalingUrl := fmt.Sprintf("%s:%s", config.SignalingUrl, config.SignalingPort)

	log.Printf("Resolving signaling server at `%s`...", fmt.Sprintf("%s/signaling/status", fullSignalingUrl))

	httpClient := http.Client{
		Timeout: time.Duration(config.IdleTimeout) * time.Millisecond,
	}

	resolveResponse, err := httpClient.Get(fmt.Sprintf("%s/signaling/status", fullSignalingUrl))
	if err != nil {
		log.Fatalln("Can't resolve signaling server: ", err) // exit
	} else if resolveResponse.StatusCode != 200 {
		log.Fatalln("Signaling server responded to status requedst with non-2xx status") // exit
	}

	log.Println("Signaling server resolved")
	log.Printf("Marking myself as ready with display name `%s`", config.DisplayName)
	readyPayloadBytes, err := json.Marshal(signaling.ClientReadyRequest{DisplayName: config.DisplayName})
	if err != nil {
		log.Fatalln("Error while forming ready request: ", err) // exit
	}

	readyResponse, err := httpClient.Post(fmt.Sprintf("%s/signaling/clients/ready", fullSignalingUrl), "application/json", bytes.NewBuffer(readyPayloadBytes))
	if err != nil {
		log.Fatalln("Error while executing ready request: ", err) // exit
	} else if readyResponse.StatusCode != 200 {
		log.Fatalln("Signaling server rejected ready request") // exit
	}

	// set up udp listener
	// TODO: replace with configurable port
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()

		conn, err := net.ListenPacket("udp", ":8585")
		if err != nil {
			log.Println("Error creating socket:", err)
			return
		}
		defer conn.Close()

		log.Println("Listening on :8585...")

		// TODO: consider safer alternatives - protobuf?
		buf := make([]byte, 1024)

		for {
			select {
			case <-exitchan:
				return
			default:
			}

			err := conn.SetReadDeadline(time.Now().Add(1_000 * time.Millisecond))
			if err != nil {
				log.Println("Error setting UDP read deadline: ", err)

				continue
			}

			_, addr, err := conn.ReadFrom(buf)
			if err != nil {
				if strings.Contains(err.Error(), "i/o timeout") {
					log.Println("UDP wait cycle timeout, re-iterating")
				} else {
					log.Println("Error reading UDP packet: ", err)
				}

				continue
			}
			req := datagram.ClientIntroRequest{}

			err = json.NewDecoder(bytes.NewReader(buf)).Decode(&req)
			if err != nil {
				log.Println("Error parsing UDP packet: ", err)

				continue
			}

			log.Println("Got new datagram: ", req)

			res, err := json.Marshal(&datagram.ClientIntroResponse{Test: "check"})
			if err != nil {
				log.Println("Error serializing datagram response: ", err)
			}

			if _, err := conn.WriteTo(res, addr); err != nil {
				log.Println("Error writing datagram: ", err)
			}
		}
	}()

	log.Println("All set up!")

	// standby
	wg.Add(1)
	go func() {
		<-exitchan
		wg.Done()
	}()
	wg.Wait()

	log.Println("Bye")
	// prompt login
	// print chats/ask for new chat
}
