package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	models "github.com/elitracy/p2pnetwork/shared"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

var (
	ServerPrivateKey wgtypes.Key
	ServerPublicKey  wgtypes.Key
)

type RegisterRequest struct {
	Name      string `json:"name"`
	PubKey    string `json:"pub_key"`
	Endpoint  string `json:"endpoint"`
	Timestamp int64  `json:"timestamp"`
	Signature string `json:"signature"`
}

type RegisterResponse struct {
	ServerPublicKey wgtypes.Key     `json:"server_public_key"`
	Peers           []models.Device `json:"peers"`
}

type PeersResponse struct {
	Peers []models.Device `json:"peers"`
}

func generateServerKeys() error {
	ServerPrivateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return err
	}
	ServerPublicKey = ServerPrivateKey.PublicKey()
	return err
}

func registrationHandler(w http.ResponseWriter, r *http.Request) {
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

	if device != nil {
		device.LastSeen = time.Now().UTC()
		err = UpsertDevice(*device)
	} else {
		err = UpsertDevice(newDevice)
	}

	if err != nil {
		http.Error(w, "failed to upsert device: "+err.Error(), http.StatusInternalServerError)
		return
	}

	peerList, err := GetAllDevices()
	if err != nil {
		http.Error(w, "failed to get peers: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp := RegisterResponse{
		ServerPublicKey: ServerPublicKey,
		Peers:           peerList,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)

}

func peersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	peerList, err := GetAllDevices()
	if err != nil {
		http.Error(w, "Failed to get peers: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp := PeersResponse{
		Peers: peerList,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
