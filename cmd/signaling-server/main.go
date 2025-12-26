package main

import (
	"encoding/json"
	"log"
	"net/http"
	"p2p-msg/internal/signaling"
	"sync"
)

func main() {
	// 1. serve http base endpoint that allows server to be resolved
	log.Println("Starting http server with PORT=8484...")

	// add signaling server resolve handler
	http.HandleFunc("/signaling/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Unsupported method", http.StatusMethodNotAllowed)

			return
		}

		log.Printf("Resolved by %s", r.RemoteAddr)

		currentServerStatus := signaling.GetCurrentServerStatus()

		w.Header().Add("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(currentServerStatus)
		if err != nil {
			http.Error(w, "Error serialiing server status", http.StatusInternalServerError)

			return
		}
	})

	// add client ready handle
	http.HandleFunc("/signaling/clients/ready", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Unsupported method", http.StatusMethodNotAllowed)

			return
		}
		log.Printf("Client `%s` requested to mark themselves as ready to accept connections", r.RemoteAddr)

		body := signaling.ClientReadyRequest{}
		err := json.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			http.Error(w, "Error deserialiing payload", http.StatusBadRequest)

			return
		}
		log.Printf("Client `%s` requested to use `%s` as their display name", r.RemoteAddr, body.DisplayName)

		address, _ := signaling.GetAddressByPeerName(body.DisplayName)
		if address != "" && address != r.RemoteAddr {
			http.Error(w, "This display name is already in use", http.StatusBadRequest)

			return
		}
		signaling.AddToActiveClientsList(body.DisplayName, r.RemoteAddr)

		log.Printf("Granted use of `%s` display name to `%s`", body.DisplayName, r.RemoteAddr)

		w.Header().Add("Content-Type", "application/json")
		json.NewEncoder(w).Encode(signaling.ClientReadyResponse{Result: "Accepted"})
	})

	// add client resolving handle
	http.HandleFunc("/signaling/clients/find", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Unsupported method", http.StatusMethodNotAllowed)

			return
		}
		log.Printf("Client `%s` requested search of other client", r.RemoteAddr)

		body := signaling.ClientResolveRequest{}
		err := json.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			http.Error(w, "Error deserialiing payload", http.StatusBadRequest)

			return
		}
		log.Printf("Client `%s` requested address of `%s` client", r.RemoteAddr, body.PeerName)

		address, err := signaling.GetAddressByPeerName(body.PeerName)
		if err != nil {
			http.Error(w, "This client is currently offline", http.StatusNotFound)

			return
		}

		log.Printf("Found entry with display name: `%s` and addr `%s`, responding to `%s`'s request", body.PeerName, address, r.RemoteAddr)

		w.Header().Add("Content-Type", "application/json")
		json.NewEncoder(w).Encode(signaling.ClientResolveResponse{DisplayName: body.PeerName, Address: address})
	})

	// add debug reset handle
	http.HandleFunc("/debug/reset", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Unsupported method", http.StatusMethodNotAllowed)

			return
		}
		log.Println("Resetting list of online clients...")

		signaling.DEBUG_ResetClients()

		log.Println("Active clients list reset")

		w.WriteHeader(204)
	})

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		err := http.ListenAndServe(":8484", nil)
		if err != nil {
			log.Fatalln("Error serving http server: ", err)
		}
		wg.Done()
	}()
	log.Println("Http server started")

	wg.Wait()
	log.Println("All background processes are finished, shutting down...")
}
