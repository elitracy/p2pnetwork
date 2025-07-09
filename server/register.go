package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// expected JSON:
//
//	{
//	  "name": "laptop-0",
//	  "pub_key": "<base64-ed25519-key>",
//	  "endpoint": "203.0.113.5:51820",
//	  "timestamp": 1720456290,
//	  "signature": "<base64-signed(timestamp)>"
//	}
type RegisterRequest struct {
	Name      string `json:"name"`
	PubKey    string `json:"pub_key"`
	Endpoint  string `json:"endpoint"`
	Timestamp int64  `json:"timestamp"`
	Signature string `json:"signature"`
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Decode public key
	pubKeyBytes, err := base64.StdEncoding.DecodeString(req.PubKey)
	if err != nil || len(pubKeyBytes) != ed25519.PublicKeySize {
		http.Error(w, "invalid public key", http.StatusBadRequest)
		return
	}

	// Decode signature
	sigBytes, err := base64.StdEncoding.DecodeString(req.Signature)
	if err != nil || len(sigBytes) != ed25519.SignatureSize {
		http.Error(w, "invalid signature", http.StatusBadRequest)
		return
	}

	// Verify signature on timestamp
	msg := []byte(fmt.Sprintf("%d", req.Timestamp))
	if !ed25519.Verify(pubKeyBytes, msg, sigBytes) {
		http.Error(w, "signature verification failed", http.StatusUnauthorized)
		return
	}

	// Save device
	mu.Lock()
	defer mu.Unlock()

	devices[req.Name] = Device{
		Name:     req.Name,
		PubKey:   req.PubKey,
		Endpoint: req.Endpoint,
		IP:       "100.64.0." + fmt.Sprint(len(devices)+1),
		LastSeen: time.Now(),
	}

	w.WriteHeader(http.StatusOK)
}
