package main

import (
	"encoding/json"
	"net/http"
)

func peerListHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	var peers []Device
	for _, dev := range devices {
		peers = append(peers, dev)
	}

	json.NewEncoder(w).Encode(peers)
}
