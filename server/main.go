package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
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
		Name:     req.Name,
		PubKey:   req.PubKey,
		Endpoint: req.Endpoint,
		IP:       host,
		Port:     port,
		LastSeen: time.Now().UTC(),
	}

	fmt.Println(newDevice)

	// look up device by pk
	device, err := GetDeviceByPubKey(newDevice.PubKey)
	if err != nil {
		http.Error(w, "failed to get device: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if device == nil {
		device, err = RegisterDevice(newDevice)
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

func main() {
	var err error

	err = initDB()
	if err != nil {
		log.Fatal("DB Error:", err.Error())
	}

	go http.HandleFunc("/register", deviceHandler)
	go http.HandleFunc("/peers", peersHandler)

	log.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
