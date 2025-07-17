package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/elitracy/p2pnetwork/shared"
)

const (
	keyringService = "meshnet"
	keyringUser    = "server_peerskey"
	envKeyName     = "MESHNET_SERVER_PEERS_KEY"
	peersFile      = "server_peers.json.enc"
)

type DeviceResponse struct {
	Name     string `json:"name"`
	PubKey   string `json:"pub_key"`
	IP       string `json:"ip"`
	LastSeen string `json:"last_seen"`
}

type RegisterRequest struct {
	Name      string `json:"name"`
	PubKey    string `json:"pub_key"`
	Endpoint  string `json:"endpoint"`
	Timestamp int64  `json:"timestamp"`
	Signature string `json:"signature"`
}

var (
	devices   = make(map[string]DeviceResponse) // map by pubkey
	devicesMu sync.Mutex
	aesKey    []byte
)

func deviceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// TODO: Verify signature and timestamp here (omitted for brevity)
	host, port, err := net.SplitHostPort(r.RemoteAddr)

	newDevice := models.Device{
		Name:      req.Name,
		PubKey:    req.PubKey,
		Endpoint:  req.Endpoint,
		IP:        host,
		Port:      port,
		LastSeen:  time.Now().UTC(),
		Connected: false,
	}

	// look up device by pk
	device, err := GetDeviceByPubKey(newDevice.PubKey)
	if err != nil {
		http.Error(w, "failed to get device: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if device == nil {
		device, err = RegisterDevice(newDevice)
	} else {
		device.LastSeen = time.Now().UTC()
		err = UpdateDevice(*device)
		if err != nil {
			http.Error(w, "failed to connect device: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if err != nil {
		http.Error(w, "failed to save device: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func peersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	peerList, err := GetAllDevices()
	if err != nil {
		http.Error(w, "failed to get peers: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(peerList)
}

func checkPeers() {
	const PING_HEARTBEAT = time.Second * 10

	for {
		peerList, err := GetAllDevices()
		if err != nil {
			log.Fatal("failed to get peers:", err.Error())
		}

		fmt.Printf("✅ Active Peers:\n")
		for _, peer := range peerList {
			if time.Since(peer.LastSeen) > PING_HEARTBEAT*3 {
				peer.Connected = false
				UpdateDevice(peer)
			}

			if peer.Connected {
				fmt.Printf("- %s @ %s (%s:%s)\n", peer.Name, peer.IP, peer.Endpoint, peer.Port)
			}
		}
		time.Sleep(PING_HEARTBEAT)
	}
}

func requestMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Grab Authorization header (or any other)
		authHeader := r.Header.Get("Authorization")
		pubkey_b64 := strings.Split(authHeader, " ")[1]

		// check if exists
		device, err := GetDeviceByPubKey(pubkey_b64)
		if err != nil {
			http.Error(w, "failed to get device: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if device != nil {
			// update connection
			device.LastSeen = time.Now().UTC()
			UpdateDevice(*device)
		}

		// Pass the request to the next handler
		next.ServeHTTP(w, r)
	})
}

func main() {
	var err error

	mux := http.NewServeMux()
	middlewareMux := requestMiddleware(mux)

	err = initDB()
	if err != nil {
		log.Fatal("DB Error:", err.Error())
	}

	go mux.HandleFunc("/device", deviceHandler)
	go mux.HandleFunc("/peers", peersHandler)

	go checkPeers()

	log.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", middlewareMux))

}
